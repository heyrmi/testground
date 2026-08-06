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
| Classic | `/classic` | Go templates, no JavaScript | shipping |
| Legacy | `/legacy` | jQuery 3.7, Bootstrap 3.4 | shipping |
| Hypermedia | `/hx` | htmx 2, Alpine.js | later |
| Realtime | `/live` | WebSocket, SSE | later |

## Challenges in v0.2.0

Twenty-two pages covering 88 distinct concepts, with 184 documented selectors
and 26 endpoints. Every one of them is in the manifest.

**Classic — `/classic`, no JavaScript at all**

| Challenge | Tier | Teaches |
|---|---|---|
| Text inputs and a full page reload | T1 | Element handles go stale across a reload; locators don't |
| Checkboxes, radios and selects | T1 | Repeated field names, multi-select APIs, disabled options |
| Six things that look like buttons | T1 | Only three post anything — role over appearance |
| Sliders, colours and native date inputs | T2 | Controls with nothing to type into; an unreachable native dialog |
| Readonly, disabled and aria-disabled | T2 | Three identical-looking states, three different behaviours |
| Redirect chains, status codes and meta refresh | T2 | 307 keeps your POST method; 303 does not |
| Error pages that are still pages | T2 | Every content assertion passes on a 500 |
| Slow to answer, and never finished | T2 | Document ready versus load, and why the default hangs |
| File uploads and the rules that are not enforced | T2 | `accept` stops nothing; size limits fail after transfer |
| Downloads and dispositions | T2 | A download is not a navigation |
| Same-origin, cross-origin and nested frames | T3 | Page script is blocked where your framework is not |

**Legacy — `/legacy`, jQuery and Bootstrap 3**

| Challenge | Tier | Teaches |
|---|---|---|
| alert, confirm, prompt and the ones that stack | T2 | Dialogs are not DOM; register the handler first |
| The dialog element, modal and not | T2 | Visible and enabled is still not clickable |
| New tabs, popups and one that writes back | T2 | A new tab is a context your locators can't see |
| pushState, replaceState and the back button | T2 | URL changes with no request and no load event |
| Debounced search that replaces its own results | T2 | Waiting for a state, not a duration |
| Six ways to be invisible | T3 | `opacity: 0` is invisible and fully clickable |

**Modern SPA — `/app`, React 19** · **Components — `/wc`, Lit**

| Challenge | Tier | Teaches |
|---|---|---|
| Element that appears after a delay | T1 | Wait for a condition, not a guessed duration |
| Toast that appears and then removes itself | T2 | Portals, transient elements, one testid matching many |
| Ten thousand rows, twenty in the DOM | T3 | Virtualisation and inner scroll containers |
| Optimistic update that reverts | T3 | A green test against a value the server discards |
| Three nested shadow roots | T3 | Shadow traversal, slots, composed events |

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
playground serve                          # loopback on :7373, second origin on :7374
playground serve --addr 0.0.0.0:7373      # expose it, for containers and workshops
playground serve --seed 1337              # different content, still deterministic
playground serve --cross-origin-addr ""   # bind one port only
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

v0.2.0 covers the ground the-internet and UI Testing Playground cover, on your
own machine and with a stability contract. What follows, in order:

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
