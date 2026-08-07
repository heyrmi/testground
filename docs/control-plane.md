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
| `POST /api/control/flake` | `{"challenge":"retries","probability":0.3}` | Makes one challenge misbehave sometimes |
| `POST /api/control/clock` | `{"action":"advance","ms":86400000}` | freeze, unfreeze, advance, set, reset |
| `POST /api/control/feature` | `{"flag":"visual-regression.diff","enabled":true}` | Turns a named flag on or off |
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

## Which challenges a flake rule reaches

A flake rule is not a generic fault. It is keyed by challenge id, and it means
one specific thing per challenge: the behaviour that page is about goes wrong
this time. These are the ids that honour one, and what each does.

| `challenge` | What a fired flake does |
|---|---|
| `optimistic-revert` | Refuses a toggle the server would have accepted, so the row flips back. Any row can be made to revert, not only the ones the tasks endpoint publishes in advance as locked. The refusal says so in its `reason` rather than claiming the row is locked. |
| `retries` | Refuses the call **after** `failFirst` is spent. The budget still behaves normally; what changes is that the endpoint no longer recovers, which is the case a retry loop with a fixed attempt count cannot ride out. |
| `data-table` | Reverses the whole result set before paging, while the response still reports the `sort` and `dir` that were asked for. Refreshing one query answers with two different orders, and the rows on a page change rather than merely swapping places. |
| `request-races` | Drops the delay the request asked for and answers at once, so whichever search was arranged to lose the race sometimes wins it. The `ms` in the response is what was actually waited, so a run says which happened. |

An id with no rule set never fires, and asking is free: a handler calls the
flake check on every request and an unset rule draws nothing from the seeded
stream at all. A page nobody has asked to misbehave is byte-identical to its
documented self, and setting a rule after a hundred requests gives the same
sequence as setting it before the first.

Anything not in this table ignores a rule naming it. `virtual-list` is the
deliberate omission: its defining behaviour is windowed rendering, which
happens in the browser, and the only thing the server could break instead is
the data — a different lesson wearing this one's name.

Every flaked response carries `X-Playground-Flaked` with the challenge id. A
refusal a page produces by design and one a rule produced are otherwise the
same status with the same shape, and a test that cannot tell them apart cannot
say which of the two it has just proved.

## Which challenges a feature flag reaches

Flags are a flat namespace, so the ones a challenge reads are named after it.

| `flag` | What turning it on does |
|---|---|
| `visual-regression.diff` | Widens the swatch by one pixel for the whole session — the same difference `?diff=1` makes, without rewriting the URLs a suite already navigates to. Either way of asking produces the same capture, so a baseline taken by one run is worth something to the other. |
| `hostile-locators.rebuild` | Ships a new build on every read of the build endpoint, so every generated class name changes with nobody having pressed anything. The button on the page ships a build when someone asks; this is the version that reaches people, where a selector stops matching between a run and its rerun. |

An unset flag is off, which is every flag until a session sets one.

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
- feature: `enabled` false

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

A flake rule asks the same question of one page rather than of the network.
Does your table test assert on the order the table is showing, or only that
some rows arrived?

```sh
curl -X POST localhost:7373/api/control/flake -H "$S" \
  -d '{"challenge":"data-table","probability":1}'
```

The response still says it sorted by name ascending. The rows are the other
way round. A test that reads `current-sort` and stops there goes green.
