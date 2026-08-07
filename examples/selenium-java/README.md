# Selenium reference suite

Worked solutions to every challenge, in Selenium 4 and JUnit 5. The companion
to [`examples/playwright-ts`](../playwright-ts): the same challenges solved with
a different tool, so the playground is not accidentally shaped around one
framework's idioms.

```sh
# from the repository root, in another terminal
go run ./cmd/playground serve

# then
cd examples/selenium-java
mvn test
```

Java 21 and Maven. Chrome is found automatically by Selenium Manager, which
also fetches a matching driver on first run — that is the only thing here that
needs the network, and only once.

| Property | Default | What it does |
|---|---|---|
| `-Dplayground.baseUrl` | `http://127.0.0.1:7373` | Where the playground is listening |
| `-Dplayground.headless` | `true` | Set `false` to watch it run |

```sh
mvn test -Dplayground.headless=false -Dtest=DelayedElementTest
```

## How a session is pinned

The Playwright suite gives each test its own copy of the playground with the
`X-Playground-Session` header. WebDriver cannot set a request header on
ordinary navigation, so this suite uses the other mechanism the server offers
and pins the same session through the `playground_session` cookie instead.
`Playground#pinSession` loads a cheap page, replaces the session the server
handed out with one named after the test class, and leaves the browser there.

The isolation that buys is not assumed. `SessionIsolationTest` proves it: it
completes a task in its own session and then reads the same endpoint from a
neighbouring one, which must still see nothing done.

## Retries are off

Deliberately, as in the Playwright suite. A playground whose behaviour is fixed
by a seed should not need them, and this suite exists partly to prove that. A
flaky test here means either the test or the challenge is wrong.

## Adding one

One class per challenge, named for it: `delayed-element` is
`DelayedElementTest`. Extend `Playground`, which gives you `open`, `find`,
`click`, `text`, `count` and the `waitFor…` family. `go test ./...` at the
repository root fails if a challenge has no class here, so the suite cannot
quietly fall behind the catalogue.

Locate by the test ids the challenge declares in its manifest entry — they are
the published contract, and `playground manifest` prints them:

```sh
go run ./cmd/playground manifest | jq '.challenges[] | select(.id=="toast") | .selectors'
```
