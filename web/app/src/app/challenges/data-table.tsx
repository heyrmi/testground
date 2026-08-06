import { useEffect, useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

interface Row {
  id: number
  name: string
  email: string
  status: string
  amount: string
  note: string
}

interface Payload {
  rows: Row[]
  total: number
  page: number
  pages: number
  sort: string
  dir: string
}

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/data-table',
  validateSearch: (search: Record<string, unknown>) => ({
    size: clampInt(search.size, 10, 1, 100),
    state: typeof search.state === 'string' ? search.state : '',
  }),
  component: DataTable,
})

const COLUMNS = ['name', 'status', 'amount'] as const

function DataTable() {
  const { size, state } = route.useSearch()
  const [payload, setPayload] = useState<Payload | null>(null)
  const [loading, setLoading] = useState(false)
  const [failed, setFailed] = useState(false)
  const [sort, setSort] = useState({ column: '', dir: 'asc' })
  const [filter, setFilter] = useState('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<number[]>([])
  const [notes, setNotes] = useState<Record<number, string>>({})
  const selectAll = useRef<HTMLInputElement>(null)

  useEffect(() => {
    let current = true
    setLoading(true)
    setFailed(false)

    const params = new URLSearchParams({
      sort: sort.column,
      dir: sort.dir,
      q: filter,
      page: String(page),
      size: String(size),
      state,
    })

    fetch(`/api/app/table/rows?${params}`)
      .then(async (res) => {
        if (!res.ok) throw new Error(String(res.status))
        return (await res.json()) as Payload
      })
      .then((body) => {
        if (!current) return
        setPayload(body)
        setLoading(false)
      })
      .catch(() => {
        if (!current) return
        setFailed(true)
        setLoading(false)
      })

    return () => {
      current = false
    }
  }, [sort, filter, page, size, state])

  const rows = payload?.rows ?? []
  const onPage = rows.map((row) => row.id)
  const chosenHere = onPage.filter((id) => selected.includes(id))

  // The third state: some but not all. It is a property rather than an
  // attribute, so it does not appear in the markup at all.
  useEffect(() => {
    if (selectAll.current) {
      selectAll.current.indeterminate = chosenHere.length > 0 && chosenHere.length < onPage.length
    }
  }, [chosenHere.length, onPage.length])

  function toggleSort(column: string) {
    setPage(1)
    setSort((current) =>
      current.column === column
        ? { column, dir: current.dir === 'asc' ? 'desc' : 'asc' }
        : { column, dir: 'asc' },
    )
  }

  return (
    <ChallengePage id="data-table">
      <div className="flex flex-wrap items-center gap-3">
        <input
          data-testid="filter"
          type="search"
          placeholder="filter by name"
          className="rounded-md border border-line bg-sunken px-2 py-1 text-sm"
          value={filter}
          onChange={(event) => {
            setPage(1)
            setFilter(event.target.value)
          }}
        />
        <span className="text-sm text-muted">
          <b data-testid="total-rows">{payload?.total ?? 0}</b> rows ·{' '}
          <b data-testid="selected-count">{selected.length}</b> selected
        </span>
        <span className="font-mono text-xs text-muted" data-testid="current-sort">
          {payload?.sort ? `${payload.sort} ${payload.dir}` : 'unsorted'}
        </span>
      </div>

      {loading && (
        <p className="mt-4 text-sm text-muted" data-testid="table-loading">
          Fetching rows…
        </p>
      )}
      {failed && (
        <p className="mt-4 text-sm" data-testid="table-error">
          The server refused. This is an outcome, not a slow success.
        </p>
      )}
      {!loading && !failed && rows.length === 0 && (
        <p className="mt-4 text-sm text-muted" data-testid="table-empty">
          Nothing matches “{filter}”.
        </p>
      )}

      {rows.length > 0 && (
        <table className="mt-4 w-full border-collapse text-sm" data-testid="table">
          <thead>
            <tr>
              <th className="w-8 border-b border-line pb-2 text-left">
                <input
                  ref={selectAll}
                  type="checkbox"
                  aria-label="Select all on this page"
                  data-testid="select-all"
                  checked={onPage.length > 0 && chosenHere.length === onPage.length}
                  onChange={(event) =>
                    setSelected((current) =>
                      event.target.checked
                        ? [...new Set([...current, ...onPage])]
                        : current.filter((id) => !onPage.includes(id)),
                    )
                  }
                />
              </th>
              {COLUMNS.map((column) => (
                <th key={column} className="border-b border-line pb-2 text-left">
                  <button
                    data-testid={`sort-${column}`}
                    className="!border-0 !bg-transparent !px-0 font-semibold"
                    onClick={() => toggleSort(column)}
                  >
                    {column}
                    {payload?.sort === column ? (payload.dir === 'asc' ? ' ↑' : ' ↓') : ''}
                  </button>
                </th>
              ))}
              <th className="border-b border-line pb-2 text-left">note</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.id} data-testid="row" data-id={row.id} className="border-b border-line">
                <td className="py-1">
                  <input
                    type="checkbox"
                    aria-label={`Select ${row.name}`}
                    data-testid="select-row"
                    checked={selected.includes(row.id)}
                    onChange={(event) =>
                      setSelected((current) =>
                        event.target.checked
                          ? [...current, row.id]
                          : current.filter((id) => id !== row.id),
                      )
                    }
                  />
                </td>
                <td className="py-1">{row.name}</td>
                <td className="py-1">{row.status}</td>
                <td className="py-1 text-right font-mono">{row.amount}</td>
                <td className="py-1">
                  <EditableNote
                    value={notes[row.id] ?? ''}
                    onCommit={(next) => setNotes((current) => ({ ...current, [row.id]: next }))}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div className="mt-4 flex items-center gap-3 text-sm">
        <button data-testid="page-prev" disabled={page <= 1} onClick={() => setPage((n) => n - 1)}>
          Previous
        </button>
        <span data-testid="page-label">
          page {payload?.page ?? 1} of {payload?.pages ?? 1}
        </span>
        <button
          data-testid="page-next"
          disabled={!payload || page >= payload.pages}
          onClick={() => setPage((n) => n + 1)}
        >
          Next
        </button>
      </div>
    </ChallengePage>
  )
}

// Commits on blur rather than on change, so a value read while the input
// still has focus has not been saved anywhere yet.
function EditableNote({ value, onCommit }: { value: string; onCommit: (next: string) => void }) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(value)

  if (!editing) {
    return (
      <span
        data-testid="cell-note"
        data-committed={value}
        className="cursor-text text-muted"
        onClick={() => {
          setDraft(value)
          setEditing(true)
        }}
      >
        {value || 'click to edit'}
      </span>
    )
  }

  return (
    <input
      autoFocus
      data-testid="cell-note-input"
      className="w-32 rounded border border-line bg-sunken px-1 text-sm"
      value={draft}
      onChange={(event) => setDraft(event.target.value)}
      onBlur={() => {
        onCommit(draft)
        setEditing(false)
      }}
      onKeyDown={(event) => {
        if (event.key === 'Enter') event.currentTarget.blur()
      }}
    />
  )
}
