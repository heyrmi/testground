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

**A control plane.** Latency, failure rates, flake probability and the clock
are all injectable per session, so you can make your own copy of the
playground misbehave without touching anyone else's. Injected chaos still
replays: it is drawn from your seed, so a test that fails against a 50%
failure rate fails the same way next run. See
[the control plane](docs/control-plane.md).

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
| Realtime | `/live` | Vanilla TypeScript, WebSocket, SSE | shipping |

## Challenges in v0.7.0

Forty-two pages covering 166 distinct concepts, with 393 documented selectors.
Every one is in the manifest, and every one can be made slow, broken or flaky
through the control plane.

**Classic — `/classic`, no JavaScript at all**

| Challenge | Tier | Teaches |
|---|---|---|
| Text inputs and a full page reload | T1 | Element handles go stale across a reload; locators don't |
| Checkboxes, radios and selects | T1 | Repeated field names, multi-select APIs, disabled options |
| Six things that look like buttons | T1 | Only three post anything — role over appearance |
| Sliders, colours and native date inputs | T2 | Controls with nothing to type into; an unreachable native dialog |
| Readonly, disabled and aria-disabled | T2 | Three identical-looking states, three different behaviours |
| Log in, and everything that guards it | T2 | CSRF, throttles, and server-side logout |
| A second factor, and a link you cannot read | T3 | Time-based codes cannot be fixtures |
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

**Modern SPA — `/app`, React 19**

| Challenge | Tier | Teaches |
|---|---|---|
| Element that appears after a delay | T1 | Wait for a condition, not a guessed duration |
| Toast that appears and then removes itself | T2 | Portals, transient elements, one testid matching many |
| Ten thousand rows, twenty in the DOM | T3 | Virtualisation and inner scroll containers |
| Optimistic update that reverts | T3 | A green test against a value the server discards |
| An endpoint that fails until it does not | T3 | Retry policy versus feature, and what retries hide |
| Responses that arrive in the wrong order | T3 | A quiet network is not a finished page |
| Six boxes pretending to be one field | T3 | One field made of six; paste is the bulk path |
| Controls that are not the elements they look like | T3 | No checkbox, no range input, no value to set |
| Elements that leave between finding and using them | T3 | Detachment, unstable ids, unmount mid-interaction |
| A modal that is not where you think it is | T3 | Portals, focus traps, scroll locks, intercepted clicks |
| A table that sorts on the server | T3 | Indeterminate is a third state; empty is not slow |
| Two kinds of dragging | T3 | Native drag is not mouse movement |
| Menus that need the pointer to stay put | T3 | A double click is also two single clicks |
| An access token that expires mid-suite | T3 | Move the clock; don't wait sixty seconds |
| A page that gives you nothing to hold on to | T4 | Generated class names, duplicate ids, div soup |
| A page heavy enough to change how your tools behave | T3 | A blocked thread stops your waiting too |
| The same page in five scripts | T2 | Prose assertions break under translation |
| A block that looks the same every time | T2 | Prove the comparison can fail before trusting it |

**Realtime — `/live`, vanilla TypeScript over WebSocket and SSE**

| Challenge | Tier | Teaches |
|---|---|---|
| A socket that talks back, and one that just talks | T2 | Updates with no triggering action to wait after |
| A connection that drops, and messages out of order | T3 | A dead socket looks exactly like a quiet one |
| A stream that ends, one that stalls, one that writes | T3 | Stalled is neither failed nor finished |

**Components — `/wc`, Lit and vanilla custom elements**

| Challenge | Tier | Teaches |
|---|---|---|
| Three nested shadow roots | T3 | Shadow traversal, slots, composed events |
| A closed shadow root, and what to do instead | T4 | Unreachable by design; properties and events are the surface |

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
playground manifest                       # print the catalogue without a server
playground seed --session worker-1        # what seed is that worker on?
playground seed 1337 --session worker-1   # put it on another one
playground version --json
```

| Endpoint | What it is |
|---|---|
| `GET /api/challenges` | Full manifest, scoped to your session |
| `GET /api/challenges/{id}` | One challenge |
| `GET /api/health` | Challenge and session counts |
| `GET /api/version` | Build identity |
| `POST /api/control/*` | [Latency, failure, flake, clock, seed, reset](docs/control-plane.md) |
| `GET /api/control/state` | Everything about the copy you are driving |

## Building from source

```sh
make        # build the frontend, then the binary
make test   # go test ./...
```

Go 1.23+ and Node 24 are the only requirements. The frontend bundle is
committed so `go install` produces a working binary; rebuild it with
`make web` whenever you touch `web/app`.

## Roadmap

v0.7.0 adds hard mode. What follows, in order:

- **v0.8.0** — accessibility and mobile, each page shipping an
  expected-axe-violations fixture so a11y tooling can be validated against
  known output. Deferred deliberately: doing it properly needs a real
  accessibility engine, and a half-built one would be worse than none.
- **v1.0.0** — composite end-to-end scenarios, docs site, Selenium reference
  suite, Docker and Homebrew.

## Contributing

New challenges are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) walks through
adding one; the short version is that a challenge is declared as data, and the
registry refuses to start if it is missing a description, a hint, a stable URL
under its zone, or its selectors.

## Licence

MIT. See [LICENSE](LICENSE).
