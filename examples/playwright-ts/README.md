# Playwright reference suite

Worked solutions for every challenge, in TypeScript. This is also the
playground's own integration suite: if a page's contract moves, these fail.

```sh
npm ci
npx playwright install --with-deps chromium
npm test
```

The config starts the server for you with `go run`, so nothing needs to be
running first. Point the suite at an already-running instance instead with:

```sh
PLAYGROUND_PORT=7373 npm test
```

## What to copy from here

**Pin a session per test.** `tests/fixtures.ts` gives every test its own
isolated copy of the playground through the `X-Playground-Session` header.
That is what lets the suite run fully parallel while tests mutate server
state.

**No retries.** `playwright.config.ts` sets `retries: 0` deliberately. The
playground is deterministic, so a flaky result here is a real defect in the
test rather than noise to be papered over.

**Read the manifest instead of hard-coding.** `manifest.spec.ts` walks
`/api/challenges` and checks every page against what it claims about itself,
so it keeps covering new challenges without being edited.

Each spec also keeps one deliberately instructive case — the guessed sleep,
the assertion that passes against a value the server is about to discard —
because seeing the wrong approach pass is the fastest way to learn why it is
wrong.
