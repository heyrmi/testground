# testground

**A testing playground that does not move under you.**

A deterministic, offline, versioned practice site for QA engineers and SDETs,
shipped as one Go binary. No network, no rate limits, no page quietly changing
and breaking the course you wrote last year.

```sh
go install github.com/heyrmi/testground/cmd/playground@latest
playground serve
```

```
testground v0.1.0 listening on http://127.0.0.1:7373 (seed 42)
```

That is the whole setup. No Node runtime, no database, no external services,
no internet connection.

---

## Why this exists

The practice sites everyone uses — Sauce Demo, the-internet, UI Testing
Playground — are jQuery-era, hosted-only, and mostly unmaintained. They lack
the constructs that actually break test suites today: shadow DOM, virtualised
lists, optimistic UI, WebSocket-driven regions, token refresh mid-suite.

They also fail operationally. They go down. They rate-limit workshops. They
change without warning. They cannot be used offline or inside an air-gapped
corporate network.

## What makes this different

**Determinism by default.** `--seed 42` produces the same page content on every
run and every machine. Randomness flows through one seeded source, split into
named streams so a challenge's output never depends on what other challenges
happened to draw first. Nothing is flaky unless you ask for flakiness.

**Session isolation.** Every client gets its own copy of the playground, keyed
on the `X-Playground-Session` header or a cookie. Parallel test workers never
see each other's mutations. This was designed in from the first commit, not
retrofitted.

**A stability contract.** Once a challenge page is released, its DOM contract
and behaviour never change. See [the contract](docs/stability-contract.md).

**Self-describing.** `GET /api/challenges` returns a machine-readable manifest
of every challenge: id, url, tier, tags, concepts, the selectors worth
locating, and the endpoints behind the page. Documentation, coverage tooling
and page-object generators read that instead of scraping the site.

**Honest about difficulty.** Challenges are graded T1 (intro) to T4 (hostile —
deliberately near-unautomatable, shipped to teach what *not* to build).

## The zones

The playground is not one app. It is several coexisting frontends under one
server, deliberately spanning twenty years of web technique, so it exercises
your tooling rather than one framework's happy path.

| Zone | Path | Technology | Status |
|---|---|---|---|
| Modern SPA | `/app` | React 19, TypeScript, Tailwind | shipping |
| Components | `/wc` | Lit web components | shipping |
| Classic | `/classic` | Go templates, no JavaScript | Phase 1 |
| Legacy | `/legacy` | jQuery 3, Bootstrap 3 | Phase 1 |
| Hypermedia | `/hx` | htmx 2, Alpine.js | Phase 1 |
| Realtime | `/live` | WebSocket, SSE | Phase 5 |

## Challenges in v0.1.0

| Challenge | Tier | Teaches |
|---|---|---|
| Element that appears after a delay<br>`/app/delayed-element` | T1 | Waiting for a condition instead of sleeping for a guess |
| Toast that appears and then removes itself<br>`/app/toast` | T2 | Portals, transient elements, one test id matching many nodes |
| Ten thousand rows, twenty in the DOM<br>`/app/virtual-list` | T3 | Virtualisation, inner scroll containers, element detachment |
| Optimistic update that reverts<br>`/app/optimistic-revert` | T3 | Settling a write before believing the DOM |
| Three nested shadow roots<br>`/wc/nested-shadow` | T3 | Shadow traversal, slots, composed events |

Every page ships with a description of what it does, why it breaks naive
automation, the selectors worth locating, and a hint disclosure with the
intended approach — concepts only, never framework-specific code.

## Using it from a test suite

Pin a session per worker and the playground stays isolated under full
parallelism:

```ts
const context = await browser.newContext({
  extraHTTPHeaders: { 'X-Playground-Session': `worker-${workerIndex}` },
})
```

The [Playwright reference suite](examples/playwright-ts) has worked solutions
for every challenge and runs with retries off, on purpose — a deterministic
playground should not need them.

```sh
cd examples/playwright-ts
npm ci && npx playwright install chromium
npm test
```

## Commands

```sh
playground serve                      # bind loopback on :7373
playground serve --addr 0.0.0.0:7373  # expose it, for containers and workshops
playground serve --seed 1337          # different content, still deterministic
playground manifest                   # print the catalogue without a server
playground version --json
```

| Endpoint | What it is |
|---|---|
| `GET /api/challenges` | Full manifest, scoped to your session |
| `GET /api/challenges/{id}` | One challenge |
| `GET /api/health` | Challenge and session counts |
| `GET /api/version` | Build identity |

## Building from source

```sh
make        # build the frontend, then the binary
make test   # go test ./...
```

Go 1.23+ and Node 24 are the only requirements. The frontend bundle is
committed so `go install` produces a working binary; rebuild it with
`make web` whenever you touch `web/app`.

## Roadmap

v0.1.0 is the walking skeleton: the architecture proven end to end with five
challenges. What follows, in order:

- **v0.2.0** — Classic, Legacy and Hypermedia zones. Basic controls, frames,
  native dialogs, windows and tabs, file upload and download. Roughly forty
  challenges.
- **v0.3.0** — the control plane. Reset, reseed, latency injection, failure
  rates, flake probability, clock manipulation. The point at which this
  becomes usable in real CI.
- **v0.4.0** — the modern SPA in full: awkward inputs, DOM instability, tables
  and data, drag and gestures.
- **v0.5.0** — authentication, including token refresh mid-suite and a
  self-hosted fake identity provider.
- **v0.6.0** — realtime.
- **v0.7.0** — accessibility and mobile, each page shipping an expected-axe-
  violations fixture so a11y tooling can be validated against known output.
- **v0.8.0** — hostile locators, performance and scale, visual regression
  targets, internationalisation.
- **v1.0.0** — composite end-to-end scenarios, docs site, Selenium reference
  suite, Docker and Homebrew.

## Contributing

New challenges are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) walks through
adding one; the short version is that a challenge is declared as data, and the
registry refuses to start if it is missing a description, a hint, a stable URL
under its zone, or its selectors.

## Licence

MIT. See [LICENSE](LICENSE).
