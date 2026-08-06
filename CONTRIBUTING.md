# Contributing

New challenges are the most useful thing you can add. This walks through one.

## Getting set up

```sh
git clone https://github.com/heyrmi/testground
cd testground
make          # builds the frontend, then the binary
./playground serve
```

Go 1.23+ and Node 24. Nothing else.

While working on the frontend, `cd web/app && npm run dev` gives hot reload on
port 5173 and proxies the API to a `playground serve` running on 7373.

## The rules a challenge must follow

The registry validates every challenge at startup and refuses to serve one
that breaks these, so you will find out immediately rather than after release:

1. A stable URL under its zone's prefix.
2. `data-testid` on every meaningful element — unless the challenge is
   *specifically* about hostile locators, in which case set
   `HostileLocators: true` and say so.
3. A description: what the page does, and why it breaks naive automation.
4. A hint with the intended approach. Concepts, never framework-specific code.
5. A manifest entry with tier, tags, concepts and the selectors worth
   locating.
6. A reference test in `examples/playwright-ts`.

Beyond validation, two rules matter just as much:

- **Nondeterminism is opt-in.** A page is never randomly flaky unless
  flakiness was asked for. Fixed delays, fixed outcomes, seeded content.
- **No cross-page dependencies.** Any subset of the challenges must be a
  useful product on its own.

## Adding a challenge

### 1. Declare it

One file per challenge in its zone package, for example
`internal/zones/app/delayed_element.go`:

```go
func delayedElement() challenge.Challenge {
	return challenge.Challenge{
		ID:       "delayed-element",
		Title:    "Element that appears after a delay",
		URL:      "/app/delayed-element",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T1,
		Category: "C. Waits & Timing",
		Summary:  "...",
		WhyHard:  "...",
		Hint:     "...",
		Tags:     []string{"waits"},
		Concepts: []string{"explicit wait"},
		Selectors: []challenge.Selector{
			{TestID: "delayed-message", Note: "Appears once the delay elapses"},
		},
		Stability: challenge.Stable,
	}
}
```

Add it to that zone's `Challenges()`.

The prose is not filler. It renders on the page and it is what the manifest
publishes, so write it for someone meeting the problem for the first time.

### 2. Build the page

**SPA zone (`/app`):** a file in `web/app/src/app/challenges/`, exporting a
`route`. Wrap the interactive part in `<ChallengePage id="your-id">` — the
chrome, description and hint come from the manifest, so you never write them
twice. Register the route in `router.tsx`.

**Server-rendered zones:** a template under `templates/pages/<zone>/`, using
`{{template "challenge-header" .}}` and `{{template "challenge-panel" .}}`.

**Needs JSON?** Add the route to that zone's `API()`, which is mounted at
`/api/<zone>`. Keep the whole challenge in one package.

### 3. Keep it deterministic

Generated content comes from the session's seeded stream, named after the
challenge so it is independent of everything else:

```go
stream := sess.RNG.Stream("your-challenge-id")
```

Mutable state goes on the session, never in a package variable:

```go
state := session.Value(sess, "your-challenge-id", func() *yourState {
	return &yourState{}
})
```

Two parallel workers must never see each other's mutations. If you find
yourself reaching for a package-level variable, that is the bug.

### 4. Make timings controllable

Anything time-dependent takes a bound from the caller, so the page can be
driven fast in a suite and slow in a demo. Declare it in `Controls` and clamp
rather than reject, so a mistyped URL still yields a working page.

### 5. Write the reference test

Add a spec to `examples/playwright-ts/tests/`. Show the approach that works,
and where it teaches something, keep one case demonstrating the approach that
looks like it works and does not.

```sh
cd examples/playwright-ts && npm test
```

Retries are off. If your test is flaky, either the test or the challenge is
wrong — do not paper over it.

### 6. Rebuild the bundle

If you touched `web/app`, run `make web` and commit `web/app/dist`. The build
output is committed so `go install` yields a working binary; CI fails if it
drifts.

## Before opening a pull request

```sh
gofmt -l .           # must print nothing
go vet ./...
go test -race ./...
make web             # then commit web/app/dist if it changed
cd examples/playwright-ts && npx tsc --noEmit && npm test
```

## Dependencies

**Go dependencies are capped at ten**, and every addition needs a
justification in the pull request. This is a tool people audit before running
it inside a corporate network. Frontend dependencies are looser but not free:
prefer the platform.

## Commit messages

Conventional commits, with a body that says what shipped and why. The
changelog is generated from them.

```
feat(challenges): add delayed element (C, T1)

A message element stays out of the DOM for three seconds and then appears.
Teaches the difference between waiting for a condition and sleeping for a
guessed duration.
```
