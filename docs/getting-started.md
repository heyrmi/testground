# Getting started

Everything the playground serves lives inside one binary: the templates, the
vendored stylesheets and scripts, the compiled frontend bundle, and the fixture
media. There is nothing to configure and no external service to reach, which is
what makes it usable inside an air-gapped network and on a laptop with the
wireless turned off.

## Install it

The shortest path, if you already have Go 1.23 or later:

```sh
go install github.com/heyrmi/testground/cmd/playground@latest
```

Otherwise download an archive for your platform from the
[releases page](https://github.com/heyrmi/testground/releases) and put
`playground` on your path. Tagged releases also publish a container image and a
Homebrew cask:

```sh
docker run --rm -p 7373:7373 -p 7374:7374 ghcr.io/heyrmi/testground:latest
brew install heyrmi/tap/testground
```

Publish both ports rather than one. The image already binds `0.0.0.0`, and the
cross-origin frame challenges embed the second port, so mapping only 7373
leaves those pages pointing at a socket nothing answers.

To build from source you need Go 1.23+ and Node 24, and nothing else:

```sh
git clone https://github.com/heyrmi/testground
cd testground
make          # builds the frontend bundle, then the binary
```

## Run it

```sh
playground serve
```

```
testground v1.0.0 listening on http://127.0.0.1:7373 (seed 42)
second origin on http://127.0.0.1:7374, for the cross-origin frame challenges
```

Two ports rather than one, because a browser decides what is same-origin from
the scheme, host and port together. The frame challenges need a genuinely
different socket to embed, so the same binary binds a second one and shares its
session store with the first. `--cross-origin-addr ""` binds only the main
port, and the challenges that need the second origin are then not registered at
all — an absent challenge rather than a broken one.

The flags worth knowing on the first day:

| Flag | Default | What it changes |
|---|---|---|
| `--addr` | `127.0.0.1:7373` | Loopback by default. A container or a workshop host wants `0.0.0.0:7373`. |
| `--cross-origin-addr` | `127.0.0.1:7374` | The second origin, or empty to bind one port only. |
| `--seed` | `42` | Every session starts from this. Different seed, different content, same determinism. |
| `--session-ttl` | 30 minutes | How long an idle session's state survives. |

## Your first challenge

Open `http://127.0.0.1:7373` and the index lists every challenge grouped by
zone. Each page carries the same three things: a description of what it does,
a statement of what breaks naive automation against it, and a hint disclosure
with the intended approach in concepts rather than in code.

A good place to start, in order of how much they will surprise you:

- `/app/delayed-element` — an element that is genuinely absent for three
  seconds. The lesson is waiting for a condition rather than for a duration.
- `/classic/text-inputs` — a form and a full page reload, which is where held
  element handles go stale and locators do not.
- `/app/optimistic-revert` — a value the client invents and the server then
  discards, so an assertion that runs straight after the click passes against a
  broken feature.

## Read the manifest instead of the page

Every challenge is declared as data, and the whole catalogue is published as
JSON. Tooling should read that rather than scraping the site.

```sh
playground manifest | jq '.count'
curl -s localhost:7373/api/challenges | jq '.challenges[] | select(.tier == "T4") | .url'
curl -s localhost:7373/api/challenges/optimistic-revert | jq '.selectors'
```

`playground manifest` needs no server. It prints the same document
`/api/challenges` serves, which makes it usable as a build step: committing its
output and diffing it in CI is how a suite finds out that a contract it depends
on has moved. The [challenge catalogue](catalogue.md) on this site is generated
from exactly that command.

## Drive it from a suite

Pin one session per worker and the playground stays isolated under full
parallelism. Every request carries the header; everything the server keeps —
state, seed, clock, injected faults — hangs off it.

```ts
const context = await browser.newContext({
  extraHTTPHeaders: { 'X-Playground-Session': `worker-${workerIndex}` },
})
```

Nothing needs resetting between workers, because no two workers share anything
to reset. Within one worker, `POST /api/control/reset` clears state and rules
and keeps the seed, so a suite picks a seed once and resets between tests
without re-picking it.

A browser that sends no header gets a session cookie instead, which is what
makes the site usable by hand without any setup.

## Make it misbehave

Determinism is the default, not the only mode. The
[control plane](control-plane.md) injects latency, failures, per-challenge
flakiness, feature flags and clock movement, all scoped to your session.

```sh
S='X-Playground-Session: worker-1'

# The next three calls to this endpoint fail, then it recovers.
curl -X POST localhost:7373/api/control/failure -H "$S" \
  -d '{"route":"/api/app/optimistic-revert/tasks","status":503,"times":3}'

# Prove the retry works. Then put it back.
curl -X POST localhost:7373/api/control/reset -H "$S"
```

The failures are drawn from your seed rather than from the system's random
source, so a test that fails against a 50% failure rate fails the same way next
run. That is the point: reproducing flakiness is a different thing from having
it.

## Run the reference suites

Both suites have a worked solution for every challenge and run with retries
switched off, on purpose — a deterministic playground should not need them.
They double as the project's own integration tests.

```sh
cd examples/playwright-ts
npm ci && npx playwright install chromium
npm test
```

The [Playwright suite](../examples/playwright-ts) is the more complete of the
two and is the one to read first; there is also a
[Selenium suite](../examples/selenium-java) in Java. Read them as worked
answers: where a challenge teaches something, the suite keeps one case
demonstrating the approach that looks like it works and does not.

## Where to go next

- [The zone model](zones.md), to decide where to practise.
- [The challenge catalogue](catalogue.md), for every page's selectors,
  endpoints and controls.
- [The control plane](control-plane.md), when the default behaviour is too well
  behaved to be interesting.
- [Contributing](../CONTRIBUTING.md), when you hit a failure mode the
  playground does not yet reproduce.
