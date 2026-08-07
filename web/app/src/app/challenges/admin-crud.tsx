import { useCallback, useEffect, useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

interface Account {
  id: string
  name: string
  email: string
  role: string
  locked: boolean
}

/** A row as the page draws it: an account, plus whatever the server has not agreed to yet. */
interface Row extends Account {
  pending?: 'creating' | 'saving'
}

/** A deleted row and the position it held, so undoing puts it back rather than appending it. */
interface Binned {
  account: Account
  index: number
}

interface WriteResponse {
  account?: Account
  accepted?: boolean
  error?: string
}

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin-crud',
  validateSearch: (search: Record<string, unknown>) => ({
    latencyMs: clampInt(search.latencyMs, 800, 0, 30_000),
    undoMs: clampInt(search.undoMs, 4000, 0, 30_000),
  }),
  component: AdminCrud,
})

/**
 * Swap the optimistic row for the one the server stored. It cannot merge by id:
 * the id is the thing that changed, which is exactly why a locator captured
 * before the answer arrived points at a row that no longer exists.
 */
function replaceTemporary(current: Row[], temporary: string, confirmed: Account): Row[] {
  if (current.some((row) => row.id === temporary)) {
    return current.map((row) => (row.id === temporary ? { ...confirmed } : row))
  }
  // Re-reading the server while the create was in flight has already taken the
  // optimistic row away, so put the confirmed one back unless that read had it.
  return current.some((row) => row.id === confirmed.id) ? current : [...current, { ...confirmed }]
}

function AdminCrud() {
  const { latencyMs, undoMs } = route.useSearch()
  const [rows, setRows] = useState<Row[]>([])
  const [roles, setRoles] = useState<string[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState('')
  const [editing, setEditing] = useState<string | null>(null)
  const [draftName, setDraftName] = useState('')
  const [draftRole, setDraftRole] = useState('')
  const [newName, setNewName] = useState('')
  const [newRole, setNewRole] = useState('viewer')
  const [bin, setBin] = useState<Binned[]>([])
  const [inFlight, setInFlight] = useState(0)
  const [rollback, setRollback] = useState('')
  const [rollbacks, setRollbacks] = useState(0)

  // The queued batch is held in a ref as well as in state because the timer
  // that sends it fires long after the render that scheduled it, and it must
  // send what is queued now rather than what was queued then.
  const binned = useRef<Binned[]>([])
  const undoTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const nextTemporary = useRef(1)

  const load = useCallback(async () => {
    const res = await fetch('/api/app/admin-crud/accounts')
    const body = (await res.json()) as { accounts: Account[]; roles: string[] }
    setRows(body.accounts.map((account) => ({ ...account })))
    setRoles(body.roles)
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  // A queued delete lives in this component, so leaving the page before the
  // window closes cancels it outright: the request never left the browser to be
  // cancelled anywhere else.
  useEffect(
    () => () => {
      if (undoTimer.current !== null) clearTimeout(undoTimer.current)
    },
    [],
  )

  function note(message: string) {
    setRollback(message)
    setRollbacks((n) => n + 1)
  }

  function stashBin(next: Binned[]) {
    binned.current = next
    setBin(next)
  }

  // Entries must arrive in ascending index order for the positions to come out
  // as they went in.
  function restore(entries: Binned[]) {
    setRows((current) => {
      const next = [...current]
      for (const entry of entries) {
        next.splice(Math.min(entry.index, next.length), 0, { ...entry.account })
      }
      return next
    })
  }

  async function sendDelete(entry: Binned) {
    setInFlight((n) => n + 1)
    const res = await fetch(`/api/app/admin-crud/accounts/${entry.account.id}?latencyMs=${latencyMs}`, {
      method: 'DELETE',
    })
    const body = (await res.json()) as WriteResponse
    setInFlight((n) => n - 1)
    if (res.ok) return

    // Refused long after the row left the page, which is why it comes back
    // rather than never having gone.
    restore([entry])
    note(`${entry.account.name} was not deleted: ${body.error ?? res.status}`)
  }

  function commitQueued() {
    if (undoTimer.current !== null) {
      clearTimeout(undoTimer.current)
      undoTimer.current = null
    }
    const batch = binned.current
    if (batch.length === 0) return

    stashBin([])
    for (const entry of batch) void sendDelete(entry)
  }

  function startDelete(ids: string[]) {
    // A second delete closes the window already open rather than queuing behind
    // it, so at most one batch is ever unsent and the toast always describes the
    // newest one.
    commitQueued()

    const wanted = new Set(ids)
    const batch: Binned[] = []
    const kept: Row[] = []
    rows.forEach((row, index) => {
      // A row the server has not confirmed carries no id the server would
      // recognise, so deleting it would ask about an account that does not
      // exist yet.
      if (wanted.has(row.id) && row.pending === undefined) {
        batch.push({
          account: {
            id: row.id,
            name: row.name,
            email: row.email,
            role: row.role,
            locked: row.locked,
          },
          index,
        })
      } else {
        kept.push(row)
      }
    })
    if (batch.length === 0) return

    // Only the rows that actually left lose their tick. A row skipped above for
    // being unconfirmed is still in the table, and deselecting it would tell
    // the reader it had been dealt with when the next bulk delete would have to
    // find it again.
    const removed = new Set(batch.map((entry) => entry.account.id))

    setRows(kept)
    setSelected((current) => current.filter((id) => !removed.has(id)))
    setEditing((current) => (current !== null && wanted.has(current) ? null : current))
    stashBin(batch)

    undoTimer.current = setTimeout(() => {
      undoTimer.current = null
      commitQueued()
    }, undoMs)
  }

  function undo() {
    if (undoTimer.current !== null) {
      clearTimeout(undoTimer.current)
      undoTimer.current = null
    }
    const batch = binned.current
    stashBin([])
    restore(batch)
  }

  async function create() {
    const name = newName.trim()
    if (name === '') return

    // The id the page invents so it has something to draw. The server issues a
    // different one, and this row stops existing the moment it does.
    const temporary = `tmp-${nextTemporary.current}`
    nextTemporary.current += 1

    setRows((current): Row[] => [
      ...current,
      { id: temporary, name, email: '', role: newRole, locked: false, pending: 'creating' },
    ])
    setNewName('')
    setInFlight((n) => n + 1)

    const res = await fetch(`/api/app/admin-crud/accounts?latencyMs=${latencyMs}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, role: newRole }),
    })
    const body = (await res.json()) as WriteResponse
    setInFlight((n) => n - 1)

    const confirmed = body.account
    if (!res.ok || !confirmed) {
      setRows((current) => current.filter((row) => row.id !== temporary))
      note(`${name} was not created: ${body.error ?? res.status}`)
      return
    }
    setRows((current) => replaceTemporary(current, temporary, confirmed))
  }

  function startEdit(row: Row) {
    setEditing(row.id)
    setDraftName(row.name)
    setDraftRole(row.role)
  }

  async function saveEdit(row: Row) {
    const name = draftName.trim()
    const role = draftRole
    const before = { name: row.name, role: row.role }

    setEditing(null)
    setRows((current) =>
      current.map((candidate): Row =>
        candidate.id === row.id ? { ...candidate, name, role, pending: 'saving' } : candidate,
      ),
    )
    setInFlight((n) => n + 1)

    const res = await fetch(`/api/app/admin-crud/accounts/${row.id}?latencyMs=${latencyMs}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, role }),
    })
    const body = (await res.json()) as WriteResponse
    setInFlight((n) => n - 1)

    const confirmed = body.account
    if (!res.ok || !confirmed) {
      setRows((current) =>
        current.map((candidate): Row =>
          candidate.id === row.id ? { ...candidate, ...before, pending: undefined } : candidate,
        ),
      )
      note(`${before.name} kept its old values: ${body.error ?? res.status}`)
      return
    }
    setRows((current) =>
      current.map((candidate): Row => (candidate.id === row.id ? { ...confirmed } : candidate)),
    )
  }

  const needle = search.trim().toLowerCase()
  const visible = rows.filter(
    (row) =>
      (needle === '' || row.name.toLowerCase().includes(needle)) &&
      (roleFilter === '' || row.role === roleFilter),
  )
  const visibleIds = visible.map((row) => row.id)
  const allVisibleChosen = visibleIds.length > 0 && visibleIds.every((id) => selected.includes(id))

  function toggleAll(checked: boolean) {
    setSelected((current) =>
      checked
        ? [...new Set([...current, ...visibleIds])]
        : current.filter((id) => !visibleIds.includes(id)),
    )
  }

  return (
    <ChallengePage id="admin-crud">
      <p className="stage__label">
        A write is answered after <b data-testid="latency-ms">{latencyMs}</b> ms, and a delete waits{' '}
        <b data-testid="undo-ms">{undoMs}</b> ms in the browser before it is sent at all
      </p>

      <div className="flex flex-wrap items-end gap-2">
        <label className="text-sm">
          New account
          <input
            data-testid="new-name"
            className="mt-1 block w-56 rounded-md border border-line bg-sunken px-2 py-1"
            placeholder="name"
            value={newName}
            onChange={(event) => setNewName(event.target.value)}
          />
        </label>
        <select
          data-testid="new-role"
          className="rounded-md border border-line bg-sunken px-2 py-1 text-sm"
          value={newRole}
          onChange={(event) => setNewRole(event.target.value)}
        >
          {roles.map((role) => (
            <option key={role} value={role}>
              {role}
            </option>
          ))}
        </select>
        <button className="primary" data-testid="create-account" onClick={() => void create()}>
          Create
        </button>
      </div>

      <div className="mt-5 flex flex-wrap items-center gap-3">
        <input
          data-testid="search"
          type="search"
          placeholder="filter by name"
          className="rounded-md border border-line bg-sunken px-2 py-1 text-sm"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
        <select
          data-testid="role-filter"
          className="rounded-md border border-line bg-sunken px-2 py-1 text-sm"
          value={roleFilter}
          onChange={(event) => setRoleFilter(event.target.value)}
        >
          <option value="">every role</option>
          {roles.map((role) => (
            <option key={role} value={role}>
              {role}
            </option>
          ))}
        </select>
        <span className="text-sm text-muted">
          <b data-testid="row-count">
            {visible.length} of {rows.length}
          </b>{' '}
          rows · <b data-testid="selected-count">{selected.length}</b> selected
        </span>
        <button
          data-testid="bulk-delete"
          disabled={selected.length === 0}
          onClick={() => startDelete(selected)}
        >
          Delete selected
        </button>
        <button data-testid="reload" onClick={() => void load()}>
          Re-read the server
        </button>
      </div>

      <table className="mt-4 w-full border-collapse text-sm">
        <thead>
          <tr>
            <th className="w-8 border-b border-line pb-2 text-left">
              <input
                type="checkbox"
                aria-label="Select every row the filters leave"
                data-testid="select-all"
                checked={allVisibleChosen}
                onChange={(event) => toggleAll(event.target.checked)}
              />
            </th>
            <th className="border-b border-line pb-2 text-left">name</th>
            <th className="border-b border-line pb-2 text-left">email</th>
            <th className="border-b border-line pb-2 text-left">role</th>
            <th className="border-b border-line pb-2 text-left">state</th>
            <th className="border-b border-line pb-2 text-right">actions</th>
          </tr>
        </thead>
        <tbody>
          {visible.map((row) => (
            <tr
              key={row.id}
              data-testid="account-row"
              data-id={row.id}
              data-locked={row.locked}
              className="border-b border-line"
            >
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
              <td className="py-1">
                {editing === row.id ? (
                  <input
                    autoFocus
                    data-testid="edit-name"
                    className="w-40 rounded border border-line bg-sunken px-1"
                    value={draftName}
                    onChange={(event) => setDraftName(event.target.value)}
                  />
                ) : (
                  <span data-testid="account-name">{row.name}</span>
                )}
              </td>
              <td className="py-1 font-mono text-xs text-muted">{row.email || '—'}</td>
              <td className="py-1">
                {editing === row.id ? (
                  <select
                    data-testid="edit-role"
                    className="rounded border border-line bg-sunken px-1"
                    value={draftRole}
                    onChange={(event) => setDraftRole(event.target.value)}
                  >
                    {roles.map((role) => (
                      <option key={role} value={role}>
                        {role}
                      </option>
                    ))}
                  </select>
                ) : (
                  <span data-testid="account-role">{row.role}</span>
                )}
              </td>
              <td className="py-1 font-mono text-xs">
                <span data-testid="account-state">{row.pending ?? 'saved'}</span>
                {row.locked && (
                  <span className="ml-2 text-muted" data-testid="account-locked">
                    locked
                  </span>
                )}
              </td>
              <td className="py-1 text-right">
                {editing === row.id ? (
                  <span className="flex justify-end gap-1">
                    <button data-testid="edit-save" onClick={() => void saveEdit(row)}>
                      Save
                    </button>
                    <button data-testid="edit-cancel" onClick={() => setEditing(null)}>
                      Cancel
                    </button>
                  </span>
                ) : (
                  <span className="flex justify-end gap-1">
                    <button
                      data-testid="row-edit"
                      disabled={row.pending !== undefined}
                      onClick={() => startEdit(row)}
                    >
                      Edit
                    </button>
                    <button
                      data-testid="row-delete"
                      disabled={row.pending !== undefined}
                      onClick={() => startDelete([row.id])}
                    >
                      Delete
                    </button>
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {bin.length > 0 && (
        <div
          role="status"
          data-testid="undo-toast"
          className="mt-4 flex items-center gap-3 rounded-lg border border-line bg-sunken px-3 py-2 text-sm"
        >
          <span className="flex-1">
            {bin.length === 1
              ? `${bin[0]?.account.name ?? ''} deleted.`
              : `${bin.length} accounts deleted.`}{' '}
            Nothing has been sent to the server yet.
          </span>
          <button data-testid="undo-delete" onClick={undo}>
            Undo
          </button>
        </div>
      )}

      <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Deletes waiting to be sent</dt>
        <dd data-testid="queued-deletes">{bin.length}</dd>
        <dt className="text-muted">Writes the server has not answered</dt>
        <dd data-testid="in-flight">{inFlight}</dd>
        <dt className="text-muted">Writes the server refused</dt>
        <dd data-testid="rollback-count">{rollbacks}</dd>
      </dl>

      {rollback && (
        <p className="mt-3 text-sm text-muted" role="status" data-testid="rollback-notice">
          {rollback}
        </p>
      )}
    </ChallengePage>
  )
}
