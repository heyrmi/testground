import { useCallback, useEffect, useState } from 'react'
import { createRoute, useNavigate } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

const FIELDS = ['title', 'owner', 'notes'] as const

type FieldName = (typeof FIELDS)[number]
type Fields = Record<FieldName, string>
type Side = 'mine' | 'theirs'
type Picks = Record<FieldName, Side>

const LABELS: Record<FieldName, string> = {
  title: 'Title',
  owner: 'Owner',
  notes: 'Notes',
}

interface RecordView {
  version: number
  fields: Fields
  updatedBy: string
  updatedAt: string
}

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/autosave',
  validateSearch: (search: Record<string, unknown>) => ({
    debounceMs: clampInt(search.debounceMs, 800, 0, 10_000),
    latencyMs: clampInt(search.latencyMs, 400, 0, 30_000),
  }),
  component: Autosave,
})

function differs(a: Fields | null, b: Fields | null): boolean {
  if (a === null || b === null) return false
  return FIELDS.some((field) => a[field] !== b[field])
}

/**
 * The merge opens on the union of the two sets of edits: a field this page
 * changed starts on mine, and a field only the other writer changed starts on
 * theirs. Opening on one whole side would make the merge a slower spelling of
 * the two buttons beside it, and the choice worth having is per field.
 */
function defaultPicks(mine: Fields, acknowledged: Fields): Picks {
  const side = (field: FieldName): Side =>
    mine[field] !== acknowledged[field] ? 'mine' : 'theirs'
  return { title: side('title'), owner: side('owner'), notes: side('notes') }
}

function assemble(picks: Picks, mine: Fields, theirs: Fields): Fields {
  const take = (field: FieldName) => (picks[field] === 'mine' ? mine[field] : theirs[field])
  return { title: take('title'), owner: take('owner'), notes: take('notes') }
}

const fieldClass =
  'mt-1 block w-full rounded-md border border-line bg-sunken px-2 py-1 text-sm'

function Autosave() {
  const { debounceMs, latencyMs } = route.useSearch()
  const navigate = useNavigate()

  // base is the record as the server last confirmed it to this page, and draft
  // is what the inputs hold. Keeping both is what lets the page tell an edit
  // nobody has sent yet from one the server has agreed to, which is the
  // distinction the indicator refuses to make.
  const [base, setBase] = useState<RecordView | null>(null)
  const [draft, setDraft] = useState<Fields | null>(null)
  const [theirs, setTheirs] = useState<RecordView | null>(null)
  // The version the refused write actually carried, which is not always the
  // one base holds: a resolution writes against theirs, and if that is refused
  // too the panel has to name the version it really tried.
  const [refusedAt, setRefusedAt] = useState(0)
  const [inFlight, setInFlight] = useState(false)
  const [saveCount, setSaveCount] = useState(0)
  const [picks, setPicks] = useState<Picks | null>(null)
  const [blocked, setBlocked] = useState(false)
  const [otherNote, setOtherNote] = useState('nobody else has written yet')

  useEffect(() => {
    let current = true
    fetch('/api/app/autosave/record')
      .then((res) => res.json() as Promise<{ record: RecordView }>)
      .then((body) => {
        if (!current) return
        setBase(body.record)
        setDraft(body.record.fields)
      })
    return () => {
      current = false
    }
  }, [])

  // The version travels with the write and is never chosen by this page. A
  // client that could name the version it was creating could overwrite a change
  // it had never seen, which is the failure optimistic concurrency exists to
  // refuse.
  const save = useCallback(
    async (fields: Fields, version: number) => {
      setInFlight(true)
      try {
        const res = await fetch(`/api/app/autosave/record?latencyMs=${latencyMs}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ version, fields }),
        })
        const body = (await res.json()) as { record: RecordView }

        if (res.status === 409) {
          setTheirs(body.record)
          setRefusedAt(version)
          // Any open merge was assembled against a record that has since moved
          // again, so keeping it would let someone save a decision they made
          // about values they never saw.
          setPicks(null)
          return
        }
        if (!res.ok) return

        setBase(body.record)
        setTheirs(null)
        setPicks(null)
        setSaveCount((n) => n + 1)
      } finally {
        setInFlight(false)
      }
    },
    [latencyMs],
  )

  const dirty = differs(draft, base?.fields ?? null)

  useEffect(() => {
    // A refused write is never retried on a timer. Resending the same stale
    // version would be refused again for as long as the page is open, and the
    // panel asking which side wins would be buried under the retries.
    if (!dirty || inFlight || theirs !== null || base === null || draft === null) return

    const fields = draft
    const version = base.version
    const timer = setTimeout(() => void save(fields, version), debounceMs)
    return () => clearTimeout(timer)
  }, [dirty, inFlight, theirs, base, draft, debounceMs, save])

  useEffect(() => {
    // Registered only while a write is in flight, so the prompt marks the one
    // moment when leaving would drop an edit the server has not acknowledged
    // rather than nagging about a page with nothing outstanding.
    if (!inFlight) return

    const guard = (event: BeforeUnloadEvent) => event.preventDefault()
    window.addEventListener('beforeunload', guard)
    return () => window.removeEventListener('beforeunload', guard)
  }, [inFlight])

  async function runOtherWriter() {
    const res = await fetch('/api/app/autosave/other-writer', { method: 'POST' })
    const body = (await res.json()) as { record: RecordView }

    // Deliberately leaves base alone. Adopting the version here would hand the
    // page knowledge a real second editor could never give it, and the next
    // autosave would succeed instead of colliding.
    setOtherNote(`another writer moved the record to version ${body.record.version}`)
  }

  if (base === null || draft === null) {
    return (
      <ChallengePage id="autosave">
        <p className="text-muted">Fetching the record…</p>
      </ChallengePage>
    )
  }

  // The indicator reports the autosave loop -- running, stopped, or nothing to
  // do -- and never whether the draft has reached the server. Dirtiness is not
  // in it, which is why it reads saved for the whole debounce window and why
  // this page has a version number as well.
  const state = inFlight ? 'saving' : theirs !== null ? 'idle' : 'saved'

  return (
    <ChallengePage id="autosave">
      <p className="stage__label">
        Typing settles after <b data-testid="debounce-ms">{debounceMs}</b> ms, and the
        server takes <b data-testid="latency-ms">{latencyMs}</b> ms to answer a write
      </p>

      <div className="grid gap-8 md:grid-cols-[1fr_18rem]">
        <section>
          <div className="flex flex-col gap-3">
            {FIELDS.map((field) => (
              <label key={field} className="text-sm">
                {LABELS[field]}
                {field === 'notes' ? (
                  <textarea
                    data-testid={`field-${field}`}
                    className={fieldClass}
                    rows={3}
                    value={draft[field]}
                    onChange={(event) => setDraft({ ...draft, [field]: event.target.value })}
                  />
                ) : (
                  <input
                    data-testid={`field-${field}`}
                    className={fieldClass}
                    value={draft[field]}
                    onChange={(event) => setDraft({ ...draft, [field]: event.target.value })}
                  />
                )}
              </label>
            ))}
          </div>

          <p className="mt-4 text-sm">
            <a
              href="/app"
              data-testid="leave-link"
              onClick={(event) => {
                event.preventDefault()
                if (inFlight) {
                  setBlocked(true)
                  return
                }
                void navigate({ to: '/' })
              }}
            >
              Leave for the zone index
            </a>
          </p>

          {blocked && (
            <p className="mt-2 text-sm" data-testid="leave-blocked">
              That navigation was refused: a write was in flight, and leaving would have
              dropped an edit the server had not acknowledged. It goes through once the
              version moves.
            </p>
          )}
        </section>

        <aside className="rounded-lg border border-line p-4">
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
            <dt className="text-muted">Indicator</dt>
            <dd role="status" data-testid="save-state">
              {state}
            </dd>
            <dt className="text-muted">Version</dt>
            <dd data-testid="record-version">{base.version}</dd>
            <dt className="text-muted">Writes accepted</dt>
            <dd data-testid="save-count">{saveCount}</dd>
            <dt className="text-muted">Last written by</dt>
            <dd data-testid="updated-by">{base.updatedBy}</dd>
          </dl>

          <button
            className="mt-4 w-full"
            data-testid="simulate-other-writer"
            onClick={() => void runOtherWriter()}
          >
            Let someone else edit it
          </button>

          <p className="mt-2 text-xs text-muted" data-testid="other-writer-note">
            {otherNote}
          </p>
        </aside>
      </div>

      {theirs !== null && (
        <section
          role="alert"
          data-testid="conflict"
          className="mt-6 rounded-lg border border-line p-4"
        >
          <h2 className="m-0 text-lg font-semibold">This record moved underneath you</h2>
          <p className="mt-2 text-sm" data-testid="conflict-versions">
            You wrote against version {refusedAt}; the server holds version {theirs.version},
            last written by {theirs.updatedBy}.
          </p>

          <div className="mt-3 flex flex-wrap gap-2">
            <button data-testid="keep-mine" onClick={() => void save(draft, theirs.version)}>
              Keep mine
            </button>
            <button
              data-testid="take-theirs"
              onClick={() => {
                setBase(theirs)
                setDraft(theirs.fields)
                setTheirs(null)
                setPicks(null)
              }}
            >
              Take theirs
            </button>
            <button
              data-testid="show-merge"
              onClick={() =>
                setPicks(picks === null ? defaultPicks(draft, base.fields) : null)
              }
            >
              {picks === null ? 'Merge field by field' : 'Hide the merge'}
            </button>
          </div>

          {picks !== null && (
            <>
              <table
                className="mt-4 w-full border-collapse text-left text-sm"
                data-testid="merge-view"
              >
                <thead>
                  <tr>
                    <th className="border-b border-line pb-2">Field</th>
                    <th className="border-b border-line pb-2">Mine</th>
                    <th className="border-b border-line pb-2">Theirs</th>
                    <th className="border-b border-line pb-2">Takes</th>
                    <th className="border-b border-line pb-2" />
                  </tr>
                </thead>
                <tbody>
                  {FIELDS.map((field) => (
                    <tr
                      key={field}
                      data-testid="merge-row"
                      data-field={field}
                      className="border-b border-line last:border-b-0"
                    >
                      <th className="py-1 font-normal text-muted">{LABELS[field]}</th>
                      <td className="py-1" data-testid="merge-mine">
                        {draft[field]}
                      </td>
                      <td className="py-1" data-testid="merge-theirs">
                        {theirs.fields[field]}
                      </td>
                      <td className="py-1 font-mono" data-testid="merge-choice">
                        {picks[field]}
                      </td>
                      <td className="py-1">
                        <button
                          data-testid="merge-pick"
                          onClick={() =>
                            setPicks({
                              ...picks,
                              [field]: picks[field] === 'mine' ? 'theirs' : 'mine',
                            })
                          }
                        >
                          Use the other
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              <button
                className="primary mt-3"
                data-testid="save-merge"
                onClick={() => {
                  const merged = assemble(picks, draft, theirs.fields)
                  setDraft(merged)
                  void save(merged, theirs.version)
                }}
              >
                Save the merge
              </button>
            </>
          )}
        </section>
      )}
    </ChallengePage>
  )
}
