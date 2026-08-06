# The control plane

Everything the playground does is deterministic until you ask otherwise. The
control plane is where you ask.

It is scoped to your session, so a worker can make its own copy of the
playground slow, broken or flaky without any other worker noticing. Nothing
here is global and nothing here needs coordinating.

```sh
curl -X POST localhost:7373/api/control/failure \
  -H 'X-Playground-Session: worker-1' \
  -d '{"route":"/api/app/*","status":503,"rate":0.5}'
```

## Chaos that replays

Injected failures are not random. Every decision is drawn from your session's
seed, indexed by how many times that rule has fired, so the same seed and the
same sequence of requests produce the same failures on every run and on every
machine.

That is the difference between a flaky playground and a playground that can
reproduce flakiness. A test that fails against a 50% failure rate on seed 42
fails the same way on the next run, which means you can debug it.

The index is per rule, so one rule's decisions never shift because a different
rule fired in between. Two challenges can be driven chaotically at the same
time without interfering.

## Endpoints

All requests carry `X-Playground-Session` to say whose copy they mean. Every
one of these answers with the full state dump, so you rarely need a second
call to confirm what happened.

| Route | Body | What it does |
|---|---|---|
| `POST /api/control/session` | — | Opens a session and returns its id |
| `DELETE /api/control/session/{id}` | — | Discards a session and everything it owned |
| `POST /api/control/reset` | — | Clears every rule and every page's state, keeping the seed |
| `POST /api/control/seed` | `{"seed":1337}` | Reseeds, discarding state derived from the old seed |
| `POST /api/control/latency` | `{"route":"/api/*","ms":800,"jitter":200}` | Delays matching requests |
| `POST /api/control/failure` | `{"route":"/api/*","status":503,"rate":0.5}` | Refuses a share of matching requests |
| `POST /api/control/failure` | `{"route":"/api/*","status":503,"times":3}` | Refuses the first three, then stops |
| `POST /api/control/flake` | `{"challenge":"toast","probability":0.3}` | Makes one challenge misbehave sometimes |
| `POST /api/control/clock` | `{"action":"advance","ms":86400000}` | freeze, unfreeze, advance, set, reset |
| `POST /api/control/feature` | `{"flag":"beta","enabled":true}` | Turns a named flag on or off |
| `GET /api/control/state` | — | The full dump: seed, clock, rules, touched challenges |

### Route patterns

A trailing `*` matches by prefix; anything else is exact. `*` on its own
matches everything. That is the whole language — it is meant to be predictable
without reading the source.

### How overlapping rules combine

**Failures** are considered in the order they were added, and the first rule
that *decides to fail* wins. A rule that declines — because its `times` is
spent, or its `rate` did not come up — does not shadow the rules after it. An
exhausted narrow rule will not silently disable a broader one.

**Latency** takes the first matching rule and stops. A request has one delay,
and composing them would make two overlapping rules quietly double it.

## What the control plane cannot touch

`/api/control` never applies its own rules to itself. A failure rule matching
every route is a reasonable thing to want, and it must not take away the
ability to remove it.

`/api/health` and `/api/version` are exempt too. They sit outside the session
middleware so that monitoring does not churn the session store, which also
puts them out of reach of session-scoped rules.

Every injected response carries `X-Playground-Injected` with the status, so a
test reading a 503 can tell one it asked for from one it did not.

## Clearing a rule

Set it back to nothing rather than deleting it:

- latency: `ms` and `jitter` both zero
- failure: a `status` below 400
- flake: `probability` of zero

`POST /api/control/reset` clears everything at once, including feature flags,
and resets the clock. It leaves the seed alone, so a suite can pick a seed once
and reset between tests without re-picking it.

## From the command line

```sh
playground seed --session worker-1          # what seed is this worker on?
playground seed 1337 --session worker-1     # put it on a different one
playground seed --session worker-1 --json   # the whole state dump
```

## A worked example

Prove your retry logic actually retries, rather than hoping:

```sh
S='X-Playground-Session: worker-1'

# The next three calls to this endpoint fail, then it recovers.
curl -X POST localhost:7373/api/control/failure -H "$S" \
  -d '{"route":"/api/app/optimistic-revert/tasks","status":503,"times":3}'

# Your test drives the page. If it passes, the retry worked.
# If it hangs, you have found out that it does not retry at all.

curl -X POST localhost:7373/api/control/reset -H "$S"
```

And prove the opposite — that your suite is not quietly retrying past real
breakage:

```sh
curl -X POST localhost:7373/api/control/failure -H "$S" \
  -d '{"route":"/api/*","status":500,"rate":0.3}'
```

If the suite still passes, it is not testing what you think it is.
