/**
 * The deliberately hostile end of the web component spectrum.
 *
 * A closed shadow root is not a security boundary and never was -- anything
 * running in the page can still reach it if it tries hard enough. What it does
 * reliably is make the component untestable through the DOM, so the only
 * supported way in is whatever API the author remembered to expose. This is
 * here to be recognised and argued against, not copied.
 */

const CLOSED_STYLE = `
  :host { display: block; border: 1px dashed var(--border, #d8d5ce); border-radius: 10px;
          padding: 1rem 1.1rem; font-family: var(--font, system-ui, sans-serif);
          color: var(--text, #1c1b19); }
  .depth { margin: 0 0 .8rem; font-family: var(--mono, monospace); font-size: .6875rem;
           letter-spacing: .08em; text-transform: uppercase; color: var(--muted, #6f6b63); }
  input { font: inherit; padding: .4rem .6rem; border: 1px solid var(--border, #d8d5ce);
          border-radius: 7px; background: var(--surface, #fff); color: inherit; width: 16rem; }
  button { font: inherit; padding: .4rem .9rem; border-radius: 7px; margin-left: .4rem;
           border: 1px solid var(--accent, #b4541e); background: var(--accent, #b4541e); color: #fff; }
  .echo { margin: .8rem 0 0; font-size: .875rem; color: var(--muted, #6f6b63); }
`

class ClosedBox extends HTMLElement {
  // Kept in a field the outside cannot reach through the DOM. The property
  // and the event below are the entire supported surface.
  #root: ShadowRoot
  #value = ''

  constructor() {
    super()
    this.#root = this.attachShadow({ mode: 'closed' })
    this.#root.innerHTML = `
      <style>${CLOSED_STYLE}</style>
      <p class="depth">Closed shadow root</p>
      <label>
        <input data-testid="closed-input" placeholder="unreachable from the page" />
      </label>
      <button data-testid="closed-submit">Send</button>
      <p class="echo" data-testid="closed-echo">nothing typed yet</p>
    `

    const input = this.#root.querySelector('input')!
    const echo = this.#root.querySelector('.echo')!

    input.addEventListener('input', () => {
      this.#value = input.value
      echo.textContent = this.#value || 'nothing typed yet'
    })

    this.#root.querySelector('button')!.addEventListener('click', () => {
      this.dispatchEvent(
        new CustomEvent('pg-closed-submit', {
          detail: { value: this.#value },
          bubbles: true,
          composed: true,
        }),
      )
    })
  }

  /** The supported way to read what is inside. */
  get value(): string {
    return this.#value
  }

  /** The supported way to write it, since nothing can reach the input. */
  set value(next: string) {
    this.#value = next
    const input = this.#root.querySelector('input')!
    const echo = this.#root.querySelector('.echo')!
    input.value = next
    echo.textContent = next || 'nothing typed yet'
  }
}

/**
 * An element that styles through ::part, which is the sanctioned way to let
 * the outside reach in without opening the whole root.
 */
class PartBox extends HTMLElement {
  constructor() {
    super()
    const root = this.attachShadow({ mode: 'open' })
    root.innerHTML = `
      <style>
        :host { display: block; }
        button { font: inherit; padding: .4rem .9rem; border-radius: 7px;
                 border: 1px solid var(--border, #d8d5ce); background: var(--surface, #fff);
                 color: inherit; }
      </style>
      <button part="trigger" data-testid="part-button">Styled from outside</button>
    `
  }
}

customElements.define('pg-closed-box', ClosedBox)
customElements.define('pg-part-box', PartBox)

// Defined late on purpose. Until this runs, <pg-late-upgrade> is an unknown
// element: no shadow root, no behaviour, and an empty box that looks like a
// component that simply failed.
const LATE_UPGRADE_MS = 1200

setTimeout(() => {
  class LateUpgrade extends HTMLElement {
    connectedCallback() {
      const root = this.attachShadow({ mode: 'open' })
      root.innerHTML = `
        <p data-testid="late-content" style="margin:0;font-family:var(--font,system-ui)">
          Upgraded ${LATE_UPGRADE_MS} ms after the page was ready.
        </p>
      `
      this.setAttribute('data-upgraded', 'true')
    }
  }
  customElements.define('pg-late-upgrade', LateUpgrade)
}, LATE_UPGRADE_MS)

// The light DOM listens for the composed event, which is the one thing that
// still crosses a closed boundary.
document.addEventListener('pg-closed-submit', (event) => {
  const { value } = (event as CustomEvent<{ value: string }>).detail
  const echo = document.querySelector('[data-testid="closed-escaped"]')
  if (echo) echo.textContent = value === '' ? '(empty)' : value
})

// The property is the whole supported surface, so the page demonstrates using
// it rather than leaving a reader to guess that it exists.
document.addEventListener('DOMContentLoaded', () => {
  const host = document.querySelector('[data-testid="closed-host"]') as ClosedBox | null
  const shown = document.querySelector('[data-testid="closed-value"]')
  if (!host || !shown) return

  document
    .querySelector('[data-testid="closed-read"]')
    ?.addEventListener('click', () => {
      shown.textContent = host.value === '' ? '(empty)' : host.value
    })

  document
    .querySelector('[data-testid="closed-write"]')
    ?.addEventListener('click', () => {
      host.value = 'written through the property'
      shown.textContent = host.value
    })
})
