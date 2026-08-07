import { Fragment, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  // No step in the path and no step in the search params. That is the honest
  // design this page teaches: the wizard's position is component state, so it
  // survives neither a reload nor the browser's back button, and there is no
  // URL a test can jump to in order to skip the walk.
  path: '/wizard',
  component: Wizard,
})

const EMPTY = {
  'account-type': '',
  email: '',
  'full-name': '',
  phone: '',
  'date-of-birth': '',
  occupation: '',
  'company-number': '',
  employees: '',
}

type Field = keyof typeof EMPTY
type Draft = Record<Field, string>
type Errors = Partial<Record<Field, string>>
type Branch = 'unchosen' | 'individual' | 'business'

interface Problem {
  field: string
  step: number
  message: string
}

const STEPS = 4
const TITLES = ['Account', 'Contact', 'Details', 'Review']

const LABELS: Record<Field, string> = {
  'account-type': 'Account type',
  email: 'Email address',
  'full-name': 'Full name',
  phone: 'Phone number',
  'date-of-birth': 'Date of birth',
  occupation: 'Occupation',
  'company-number': 'Company number',
  employees: 'Employees',
}

// Step three asks an individual and a business different questions, which is
// why a locator written against one branch finds nothing on the other and
// reports a missing element rather than an answer three steps back being wrong.
function fieldsOn(step: number, branch: Branch): Field[] {
  switch (step) {
    case 1:
      return ['account-type', 'email']
    case 2:
      return ['full-name', 'phone']
    case 3:
      return branch === 'business'
        ? ['company-number', 'employees']
        : ['date-of-birth', 'occupation']
    default:
      return []
  }
}

const EMAIL_SHAPE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const DATE_SHAPE = /^\d{4}-\d{2}-\d{2}$/
const COMPANY_SHAPE = /^\d{8}$/
const WHOLE_NUMBER = /^\d+$/

const digits = (value: string) => value.replace(/\D/g, '').length

/**
 * What the page checks, which is deliberately less than what the server checks.
 * Shapes here, meanings there: an address is well formed here and at an
 * acceptable domain there, a date of birth is well formed here and old enough
 * there. Every one of those gaps is a refusal that arrives steps away from the
 * box that caused it.
 */
function problemsOn(step: number, draft: Draft, branch: Branch): Errors {
  const found: Errors = {}

  if (step === 1) {
    if (branch === 'unchosen') {
      found['account-type'] = 'choose whether this is an individual or a business'
    }
    if (!EMAIL_SHAPE.test(draft.email.trim())) {
      found.email = 'that is not an email address'
    }
  }

  if (step === 2) {
    if (draft['full-name'].trim().length < 2) found['full-name'] = 'a full name is required'
    if (digits(draft.phone) < 7) found.phone = 'a phone number needs at least seven digits'
  }

  if (step === 3 && branch === 'business') {
    if (!COMPANY_SHAPE.test(draft['company-number'].trim())) {
      found['company-number'] = 'a company number is exactly eight digits'
    }
    const employees = draft.employees.trim()
    if (!WHOLE_NUMBER.test(employees) || Number(employees) < 1) {
      found.employees = 'a business needs at least one employee'
    }
  }

  if (step === 3 && branch !== 'business') {
    if (!DATE_SHAPE.test(draft['date-of-birth'].trim())) {
      found['date-of-birth'] = 'a date of birth is required, as YYYY-MM-DD'
    }
    if (draft.occupation.trim() === '') found.occupation = 'an occupation is required'
  }

  return found
}

function Wizard() {
  const [step, setStep] = useState(1)
  const [draft, setDraft] = useState<Draft>(EMPTY)
  // Nothing is ever taken out of this. A step that passed once counts as passed
  // for the rest of the flow, even after the answer it depended on has changed,
  // which is what lets a jump forward skip a step that is no longer valid.
  const [cleared, setCleared] = useState<number[]>([])
  const [errors, setErrors] = useState<Errors>({})
  const [refusal, setRefusal] = useState('')
  const [problems, setProblems] = useState<Problem[]>([])
  const [reference, setReference] = useState('')

  const branch: Branch =
    draft['account-type'] === 'individual' || draft['account-type'] === 'business'
      ? draft['account-type']
      : 'unchosen'

  function setField(field: Field, value: string) {
    setDraft((current) => ({ ...current, [field]: value }))
  }

  // Errors belong to the step that produced them, so they leave with it.
  function goTo(next: number) {
    setErrors({})
    setRefusal('')
    setProblems([])
    setStep(next)
  }

  // The server hears about a step only here, when the page has validated it and
  // is moving on. Going back tells it nothing, and neither does jumping through
  // the progress links -- which is how a wizard's own idea of an application
  // drifts from the one that will actually be processed.
  async function advance() {
    const found = problemsOn(step, draft, branch)
    if (Object.keys(found).length > 0) {
      setErrors(found)
      return
    }

    const values: Record<string, string> = {}
    for (const field of fieldsOn(step, branch)) values[field] = draft[field]

    await fetch('/api/app/wizard/draft', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ step, values }),
    })

    setCleared((current) => (current.includes(step) ? current : [...current, step]))
    goTo(step + 1)
  }

  async function submit() {
    const res = await fetch('/api/app/wizard/submit', { method: 'POST' })
    const body = (await res.json()) as {
      error?: string
      problems?: Problem[]
      reference?: string
    }

    if (!res.ok) {
      setRefusal(body.error ?? 'the application was refused')
      setProblems(body.problems ?? [])
      return
    }

    setRefusal('')
    setProblems([])
    setReference(body.reference ?? '')
  }

  function startAgain() {
    setDraft(EMPTY)
    setCleared([])
    setReference('')
    goTo(1)
  }

  if (reference) {
    return (
      <ChallengePage id="wizard">
        <section data-testid="confirmation">
          <h2 className="m-0 text-lg font-semibold">Application received</h2>
          <p className="mt-2">
            Your reference is{' '}
            <b className="font-mono" data-testid="reference">
              {reference}
            </b>
            . The draft it was made from has been spent, so submitting again from a stale page
            fails rather than lodging it twice.
          </p>
          <button className="mt-4" data-testid="start-again" onClick={startAgain}>
            Start another application
          </button>
        </section>
      </ChallengePage>
    )
  }

  // A step is reachable once the one before it has been cleared. Clearing is
  // never undone, so this stays true after an answer changes underneath it.
  const reachable = (n: number) => n === 1 || cleared.includes(n - 1)
  const reviewed = [...fieldsOn(1, branch), ...fieldsOn(2, branch), ...fieldsOn(3, branch)]

  return (
    <ChallengePage id="wizard">
      <p className="stage__label">
        <b data-testid="step-counter">
          Step {step} of {STEPS}
        </b>{' '}
        · branch <b data-testid="branch">{branch}</b>
      </p>

      <ol className="mt-2 flex list-none flex-wrap gap-2 p-0">
        {[1, 2, 3, 4].map((n) => (
          <li key={n}>
            <button
              data-testid="step-link"
              data-step={n}
              aria-current={n === step ? 'step' : undefined}
              disabled={!reachable(n)}
              onClick={() => goTo(n)}
            >
              {n}. {TITLES[n - 1]}
            </button>
          </li>
        ))}
      </ol>

      <section className="mt-6 max-w-lg">
        {step === 1 && (
          <>
            <label className="block text-sm">
              {LABELS['account-type']}
              <select
                data-testid="account-type"
                className="mt-1 block w-72 rounded-md border border-line bg-sunken px-2 py-1"
                value={draft['account-type']}
                onChange={(event) => setField('account-type', event.target.value)}
              >
                <option value="">choose one</option>
                <option value="individual">individual</option>
                <option value="business">business</option>
              </select>
              {errors['account-type'] && (
                <FieldError field="account-type" message={errors['account-type']} />
              )}
            </label>

            <TextField
              field="email"
              placeholder="you@example.test"
              value={draft.email}
              error={errors.email}
              onChange={setField}
            />

            <p className="mt-3 text-xs text-muted">
              Addresses at <code>rejected.test</code> and <code>disposable.test</code> are refused
              — but not here. The domain is only looked at when the application is submitted.
            </p>
          </>
        )}

        {step === 2 &&
          fieldsOn(2, branch).map((field) => (
            <TextField
              key={field}
              field={field}
              value={draft[field]}
              error={errors[field]}
              onChange={setField}
            />
          ))}

        {step === 3 && (
          <>
            <p className="m-0 text-sm text-muted">
              These are the questions for {branch === 'business' ? 'a business' : 'an individual'}.
              Step one decides which set exists at all.
            </p>
            {fieldsOn(3, branch).map((field) => (
              <TextField
                key={field}
                field={field}
                type={field === 'employees' ? 'number' : 'text'}
                placeholder={field === 'date-of-birth' ? 'YYYY-MM-DD' : undefined}
                value={draft[field]}
                error={errors[field]}
                onChange={setField}
              />
            ))}
          </>
        )}

        {step === 4 && (
          <>
            <h2 className="m-0 text-lg font-semibold">Review</h2>
            <p className="mt-1 text-sm text-muted">
              This is the page's copy of the application. Ask <code>/api/app/wizard/draft</code>{' '}
              what the server has.
            </p>

            <dl className="mt-3 grid grid-cols-[10rem_1fr] gap-y-1 text-sm" data-testid="review">
              {reviewed.map((field) => (
                <Fragment key={field}>
                  <dt className="text-muted">{LABELS[field]}</dt>
                  <dd data-testid="review-value" data-field={field}>
                    {draft[field] || 'not answered'}
                  </dd>
                </Fragment>
              ))}
            </dl>

            {refusal && (
              <div className="mt-4 text-sm" data-testid="submit-error">
                <p className="m-0">{refusal}</p>
                <ul className="mt-2 list-none p-0">
                  {problems.map((problem) => (
                    <li
                      key={problem.field}
                      data-testid="problem"
                      data-field={problem.field}
                      data-step={problem.step}
                    >
                      step {problem.step}: {problem.message}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <button className="primary mt-4" data-testid="submit" onClick={submit}>
              Submit application
            </button>
          </>
        )}
      </section>

      <div className="mt-6 flex gap-2">
        {step > 1 && (
          <button data-testid="back" onClick={() => goTo(step - 1)}>
            Back
          </button>
        )}
        {step < STEPS && (
          <button className="primary" data-testid="next" onClick={advance}>
            Next
          </button>
        )}
      </div>
    </ChallengePage>
  )
}

function TextField({
  field,
  value,
  error,
  onChange,
  placeholder,
  type = 'text',
}: {
  field: Field
  value: string
  error?: string
  onChange: (field: Field, value: string) => void
  placeholder?: string
  type?: string
}) {
  return (
    <label className="mt-3 block text-sm">
      {LABELS[field]}
      <input
        data-testid={field}
        type={type}
        placeholder={placeholder}
        className="mt-1 block w-72 rounded-md border border-line bg-sunken px-2 py-1"
        value={value}
        onChange={(event) => onChange(field, event.target.value)}
      />
      {error && <FieldError field={field} message={error} />}
    </label>
  )
}

// data-field carries the same string as the box's data-testid and as the key
// the server keeps it under, so a refusal from four steps away can be traced to
// the box that caused it without a translation table in between.
function FieldError({ field, message }: { field: Field; message: string }) {
  return (
    <span className="mt-1 block text-xs" data-testid="field-error" data-field={field}>
      {message}
    </span>
  )
}
