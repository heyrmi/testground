import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/modal-portal',
  component: ModalPortal,
})

function ModalPortal() {
  const [open, setOpen] = useState(false)
  const [outcome, setOutcome] = useState('nothing yet')
  const [backgroundClicks, setBackgroundClicks] = useState(0)

  return (
    <ChallengePage id="modal-portal">
      <div className="flex flex-wrap gap-2">
        <button className="primary" data-testid="open-modal" onClick={() => setOpen(true)}>
          Open the modal
        </button>
        <button data-testid="background-button" onClick={() => setBackgroundClicks((n) => n + 1)}>
          Background button
        </button>
      </div>

      <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Modal closed with</dt>
        <dd data-testid="modal-outcome">{outcome}</dd>
        <dt className="text-muted">Background clicks</dt>
        <dd data-testid="background-clicks">{backgroundClicks}</dd>
        <dt className="text-muted">Body scroll</dt>
        <dd data-testid="scroll-state">{open ? 'locked' : 'free'}</dd>
      </dl>

      <p className="mt-4 text-sm text-muted">
        A long block follows so the page scrolls. While the modal is open the body cannot, which
        is a state a test can read and a user can feel.
      </p>
      <div className="mt-3 h-[120vh] rounded-lg border border-line bg-sunken" aria-hidden="true" />

      {open && (
        <Modal
          onClose={(how) => {
            setOutcome(how)
            setOpen(false)
          }}
        />
      )}
    </ChallengePage>
  )
}

function Modal({ onClose }: { onClose: (how: string) => void }) {
  const dialog = useRef<HTMLDivElement>(null)

  // Scroll lock and focus trap, both hand-rolled, because that is how they
  // arrive in most applications and both are observable from a test.
  useEffect(() => {
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.body.dataset.scrollLocked = 'true'

    dialog.current?.querySelector<HTMLElement>('[data-testid="modal-confirm"]')?.focus()

    return () => {
      document.body.style.overflow = previous
      delete document.body.dataset.scrollLocked
    }
  }, [])

  function onKeyDown(event: React.KeyboardEvent) {
    if (event.key === 'Escape') {
      onClose('escape')
      return
    }
    if (event.key !== 'Tab') return

    const focusable = dialog.current?.querySelectorAll<HTMLElement>('button')
    if (!focusable?.length) return

    const first = focusable[0]!
    const last = focusable[focusable.length - 1]!
    const target = event.shiftKey ? first : last
    if (document.activeElement === target) {
      event.preventDefault()
      ;(event.shiftKey ? last : first).focus()
    }
  }

  return createPortal(
    <div
      data-testid="modal-overlay"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose('overlay')
      }}
    >
      <div
        ref={dialog}
        role="dialog"
        aria-modal="true"
        aria-label="Confirm"
        data-testid="modal"
        onKeyDown={onKeyDown}
        className="w-[26rem] max-w-[90vw] rounded-xl border border-line bg-surface p-6"
      >
        <h2 className="m-0 text-lg font-semibold">Are you sure?</h2>
        <p className="mt-2 text-sm text-muted">
          This dialog is a child of body, not of the application root. Focus cannot leave it with
          Tab, and the page behind it cannot scroll.
        </p>
        <div className="mt-5 flex gap-2">
          <button className="primary" data-testid="modal-confirm" onClick={() => onClose('confirmed')}>
            Confirm
          </button>
          <button data-testid="modal-cancel" onClick={() => onClose('cancelled')}>
            Cancel
          </button>
        </div>
      </div>
    </div>,
    document.body,
  )
}
