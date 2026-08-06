import { useEffect, useOptimistic, useState, useTransition } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

interface Task {
  id: number
  title: string
  done: boolean
  rejects: boolean
}

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/optimistic-revert',
  validateSearch: (search: Record<string, unknown>) => ({
    latencyMs: clampInt(search.latencyMs, 800, 0, 30_000),
  }),
  component: OptimisticRevert,
})

function OptimisticRevert() {
  const { latencyMs } = route.useSearch()
  const [tasks, setTasks] = useState<Task[]>([])
  const [saving, setSaving] = useState<number[]>([])
  const [refused, setRefused] = useState<Task | null>(null)
  const [refusals, setRefusals] = useState(0)
  const [, startTransition] = useTransition()

  // The optimistic overlay only survives while its transition is pending. When
  // the server answers and the real state lands, anything the client invented
  // and the server did not confirm disappears on its own.
  const [shown, flipOptimistically] = useOptimistic(tasks, (current, id: number) =>
    current.map((task) => (task.id === id ? { ...task, done: !task.done } : task)),
  )

  useEffect(() => {
    let current = true
    fetch('/api/app/optimistic-revert/tasks')
      .then((res) => res.json() as Promise<{ tasks: Task[] }>)
      .then((body) => {
        if (current) setTasks(body.tasks)
      })
    return () => {
      current = false
    }
  }, [])

  function toggle(task: Task) {
    setSaving((current) => [...current, task.id])

    startTransition(async () => {
      flipOptimistically(task.id)

      const res = await fetch(
        `/api/app/optimistic-revert/tasks/${task.id}/toggle?latencyMs=${latencyMs}`,
        { method: 'POST' },
      )
      const body = (await res.json()) as { task: Task; accepted: boolean }

      startTransition(() => {
        setTasks((current) => current.map((t) => (t.id === body.task.id ? body.task : t)))
        setSaving((current) => current.filter((id) => id !== task.id))
        if (!body.accepted) {
          setRefused(body.task)
          setRefusals((n) => n + 1)
        }
      })
    })
  }

  return (
    <ChallengePage id="optimistic-revert">
      <p className="stage__label">
        The server answers after <b data-testid="latency-ms">{latencyMs}</b> ms
      </p>

      <ul className="m-0 list-none p-0">
        {shown.map((task) => {
          const inFlight = saving.includes(task.id)
          return (
            <li
              key={task.id}
              data-testid="task"
              data-task-id={task.id}
              className="flex items-center gap-3 border-b border-line py-2 last:border-b-0"
            >
              <button
                data-testid="task-toggle"
                aria-pressed={task.done}
                onClick={() => toggle(task)}
                className="w-24 shrink-0"
              >
                {task.done ? 'Undo' : 'Do it'}
              </button>
              <span className="flex-1" data-testid="task-title">
                {task.title}
              </span>
              {inFlight && (
                <span className="text-xs text-muted" data-testid="task-saving">
                  saving…
                </span>
              )}
              <span className="w-16 text-right font-mono text-xs" data-testid="task-state">
                {task.done ? 'done' : 'todo'}
              </span>
            </li>
          )
        })}
      </ul>

      <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Toggles refused</dt>
        <dd data-testid="rejected-count">{refusals}</dd>
      </dl>

      {refused && (
        <p className="mt-3 text-sm text-muted" role="status" data-testid="revert-notice">
          The server refused “{refused.title}” and the row reverted.
        </p>
      )}
    </ChallengePage>
  )
}
