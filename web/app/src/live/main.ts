/**
 * Zone 6, in plain TypeScript with nothing between the page and the socket.
 *
 * Every page here publishes its connection state and its counters as text,
 * because the useful assertion is almost never "is this message on screen" --
 * it is "has this connection reached the state I expect", and a page that only
 * renders messages leaves a test guessing.
 */

interface Message {
  seq: number
  kind: string
  text: string
  at: string
}

const $ = (id: string) => document.querySelector<HTMLElement>(`[data-testid="${id}"]`)

function set(id: string, value: string | number) {
  const node = $(id)
  if (node) node.textContent = String(value)
}

function socketURL(path: string, params: Record<string, string> = {}): string {
  const url = new URL(path, window.location.href)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  for (const [key, value] of Object.entries(params)) url.searchParams.set(key, value)
  return url.toString()
}

function search(name: string, fallback: string): string {
  return new URLSearchParams(window.location.search).get(name) ?? fallback
}

function appendLog(text: string) {
  const log = $('message-log')
  if (!log) return
  const line = document.createElement('li')
  line.dataset.testid = 'log-line'
  line.textContent = text
  log.appendChild(line)
}

/* Echo and ticker ---------------------------------------------------------- */

function wireWebSocketPage() {
  if (!$('echo-connect')) return

  let echo: WebSocket | null = null
  let echoes = 0

  $('echo-connect')?.addEventListener('click', () => {
    echo?.close()
    set('echo-state', 'connecting')
    echo = new WebSocket(socketURL('/api/live/echo'))

    echo.addEventListener('open', () => set('echo-state', 'open'))
    echo.addEventListener('close', () => set('echo-state', 'closed'))
    echo.addEventListener('message', (event) => {
      const message = JSON.parse(event.data as string) as Message
      if (message.kind === 'open') return
      echoes += 1
      set('echo-last', message.text)
      set('echo-count', echoes)
      appendLog(message.text)
    })
  })

  $('echo-send')?.addEventListener('click', () => {
    const input = $('echo-input') as HTMLInputElement | null
    if (echo?.readyState === WebSocket.OPEN && input) echo.send(JSON.stringify(input.value))
  })

  let ticker: WebSocket | null = null
  let ticks = 0

  $('ticker-connect')?.addEventListener('click', () => {
    ticker?.close()
    ticks = 0
    set('ticker-count', 0)
    set('ticker-state', 'connecting')

    ticker = new WebSocket(
      socketURL('/api/live/ticker', { ms: search('ms', '500'), count: search('count', '0') }),
    )
    ticker.addEventListener('open', () => set('ticker-state', 'open'))
    ticker.addEventListener('close', () => set('ticker-state', 'closed'))
    ticker.addEventListener('message', (event) => {
      const message = JSON.parse(event.data as string) as Message
      ticks += 1
      set('ticker-count', ticks)
      set('ticker-last-seq', message.seq)
      appendLog(`#${message.seq} ${message.text}`)
    })
  })

  $('ticker-stop')?.addEventListener('click', () => ticker?.close())
}

/* Reconnects and ordering -------------------------------------------------- */

function wireReconnectPage() {
  if (!$('flaky-connect')) return

  let socket: WebSocket | null = null
  let drops = 0
  let generation = 0
  let received = 0
  let wanted = false
  let backoff = 100

  function connect() {
    generation += 1
    set('flaky-generation', generation)
    set('flaky-state', 'connecting')

    socket = new WebSocket(
      socketURL('/api/live/flaky', { dropAfterMs: search('dropAfterMs', '2000') }),
    )

    socket.addEventListener('open', () => {
      set('flaky-state', 'open')
      backoff = 100
    })

    socket.addEventListener('message', () => {
      received += 1
      set('flaky-count', received)
    })

    // A close is the only notice. Nothing in the rendered messages changes,
    // which is why the state and the generation are published separately.
    socket.addEventListener('close', () => {
      drops += 1
      set('flaky-drops', drops)

      if (!wanted) {
        set('flaky-state', 'closed')
        return
      }
      set('flaky-state', 'reconnecting')
      window.setTimeout(connect, backoff)
      backoff = Math.min(backoff * 2, 2000)
    })
  }

  $('flaky-connect')?.addEventListener('click', () => {
    wanted = true
    drops = 0
    received = 0
    generation = 0
    backoff = 100
    set('flaky-drops', 0)
    set('flaky-count', 0)
    connect()
  })

  $('flaky-stop')?.addEventListener('click', () => {
    wanted = false
    socket?.close()
  })

  $('shuffled-connect')?.addEventListener('click', () => {
    const arrival: number[] = []
    const shuffled = new WebSocket(
      socketURL('/api/live/shuffled', { count: search('count', '6') }),
    )

    shuffled.addEventListener('message', (event) => {
      const message = JSON.parse(event.data as string) as Message
      arrival.push(message.seq)

      // Both orders, side by side, so the difference is visible rather than
      // something a reader has to take on trust.
      set('arrival-order', arrival.join(', '))
      set('sorted-order', [...arrival].sort((a, b) => a - b).join(', '))
    })

    // Added rather than unhidden: elsewhere in the playground a transient
    // element is absent from the DOM, and a hidden one would be a different
    // contract wearing the same name.
    shuffled.addEventListener('close', () => {
      const outcome = $('shuffled-outcome')
      if (!outcome || outcome.childElementCount > 0) return
      const done = document.createElement('p')
      done.dataset.testid = 'shuffled-done'
      done.textContent = 'Every message has arrived.'
      outcome.appendChild(done)
    })
  })
}

/* Server-sent events ------------------------------------------------------- */

function wireEventStreamPage() {
  if (!$('events-start')) return

  function stream(
    button: string,
    path: string,
    params: Record<string, string>,
    stateId: string,
    onEvent: (data: unknown) => void,
    onDone?: () => void,
  ) {
    $(button)?.addEventListener('click', () => {
      set(stateId, 'streaming')
      const url = new URL(path, window.location.href)
      for (const [key, value] of Object.entries(params)) url.searchParams.set(key, value)

      const source = new EventSource(url)
      source.addEventListener('update', (event) => onEvent(JSON.parse(event.data)))
      source.addEventListener('token', (event) => onEvent(JSON.parse(event.data)))

      // Only a stream that finishes sends this. The stalled one never does,
      // which is exactly why the state is worth publishing.
      source.addEventListener('done', () => {
        set(stateId, 'done')
        onDone?.()
        source.close()
      })
    })
  }

  let updates = 0
  stream('events-start', '/api/live/events', { count: search('count', '5'), ms: search('ms', '200') },
    'events-state', () => set('events-count', ++updates))

  let stalled = 0
  stream('stall-start', '/api/live/stall', { before: search('before', '3'), ms: search('ms', '150') },
    'stall-state', () => set('stall-count', ++stalled))

  let tokens = 0
  let text = ''
  stream('stream-start', '/api/live/stream', { ms: search('ms', '60') }, 'stream-state',
    (data) => {
      const { token } = data as { token: string }
      tokens += 1
      text += token
      set('stream-tokens', tokens)
      set('stream-text', text.trim())
    })
}

wireWebSocketPage()
wireReconnectPage()
wireEventStreamPage()
