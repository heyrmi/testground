import { useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/otp-input',
  component: OtpInput,
})

const LENGTH = 6
const CORRECT = '314159'

function OtpInput() {
  const [digits, setDigits] = useState<string[]>(Array(LENGTH).fill(''))
  const [verdict, setVerdict] = useState('nothing entered')
  const boxes = useRef<(HTMLInputElement | null)[]>([])

  const code = digits.join('')

  function write(next: string[]) {
    setDigits(next)
    const joined = next.join('')
    if (joined.length < LENGTH || next.some((d) => d === '')) {
      setVerdict('incomplete')
      return
    }
    setVerdict(joined === CORRECT ? 'accepted' : 'rejected')
  }

  function onChange(index: number, raw: string) {
    // A box holds one character. Typing over a filled box replaces it, and a
    // bulk write into one box keeps a single digit and drops the rest, which
    // is what makes the obvious approach fail in a way that looks like the
    // page truncating rather than the test misfiring.
    const digit = raw.replace(/\D/g, '').slice(-1)
    const next = [...digits]
    next[index] = digit
    write(next)

    if (digit && index < LENGTH - 1) boxes.current[index + 1]?.focus()
  }

  function onKeyDown(index: number, event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Backspace' && digits[index] === '' && index > 0) {
      boxes.current[index - 1]?.focus()
    }
    if (event.key === 'ArrowLeft' && index > 0) boxes.current[index - 1]?.focus()
    if (event.key === 'ArrowRight' && index < LENGTH - 1) boxes.current[index + 1]?.focus()
  }

  // Pasting distributes across the boxes, which is the one path that fills
  // them all without six separate keystrokes.
  function onPaste(event: React.ClipboardEvent<HTMLInputElement>) {
    const pasted = event.clipboardData.getData('text').replace(/\D/g, '').slice(0, LENGTH)
    if (!pasted) return

    event.preventDefault()
    const next = Array(LENGTH).fill('')
    for (const [i, character] of [...pasted].entries()) next[i] = character
    write(next)
    boxes.current[Math.min(pasted.length, LENGTH - 1)]?.focus()
  }

  return (
    <ChallengePage id="otp-input">
      <p className="stage__label">
        The code is <b data-testid="expected-code">{CORRECT}</b>, printed here so the exercise is
        the typing rather than the guessing
      </p>

      <div className="flex gap-2" data-testid="otp-group">
        {digits.map((digit, index) => (
          <input
            key={index}
            ref={(node) => {
              boxes.current[index] = node
            }}
            data-testid={`otp-${index}`}
            data-index={index}
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={1}
            className="h-12 w-10 rounded-lg border border-line bg-sunken text-center font-mono text-lg"
            value={digit}
            onChange={(event) => onChange(index, event.target.value)}
            onKeyDown={(event) => onKeyDown(index, event)}
            onPaste={onPaste}
          />
        ))}
      </div>

      <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Assembled code</dt>
        <dd data-testid="otp-value">{code || '(empty)'}</dd>
        <dt className="text-muted">Verdict</dt>
        <dd data-testid="otp-verdict">{verdict}</dd>
      </dl>

      <button className="mt-4" data-testid="otp-clear" onClick={() => write(Array(LENGTH).fill(''))}>
        Clear
      </button>
    </ChallengePage>
  )
}
