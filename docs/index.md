# The testing playground

testground is a practice site for the people who write test automation. It runs
from one Go binary on your own machine, with no network, no database and no
account, and it is built around a single idea: **a page you practise against, or
teach from, or assert on in CI, must not move under you.**

<!--generated:catalogue-stats-->

The pages are not a gallery of widgets. Each one reproduces a specific way that
real applications defeat naive automation — an optimistic update the server
discards, a list of ten thousand rows with twenty in the DOM, a token that
expires halfway through a suite, a shadow root you cannot reach into — and each
one states, on the page and in the manifest, what it does and why it is hard.

This site is the manual. The
[README](../README.md) is the shop window: it says what the project is and how
to install it in a screen and a half. Everything that needs more than that is
here, and the two deliberately do not repeat each other.

## Determinism

Nothing is random unless you ask for randomness. A seed produces the same
content on every run and every machine, and the seed is a flag:
`playground serve --seed 1337`.

Randomness flows through one seeded source, split into named streams derived
from the seed and the stream name. That last detail is what makes the promise
survive contact with a real suite: a challenge's output never depends on how
many values some other challenge happened to draw first, so adding a page in a
later version does not change the content of the pages beside it.

Time is injectable for the same reason. No handler calls the system clock, so a
token expiry is something you advance rather than something you wait for.

## Session isolation

Every client gets its own copy of the playground, keyed on the
`X-Playground-Session` header or a cookie. State, seed, clock and injected
faults all belong to a session and to nothing wider.

The consequence worth caring about is that parallel workers cannot interfere.
Worker 3 can freeze its clock, set a 50% failure rate and empty a shopping cart
while worker 4 runs the untouched, documented version of the same page. There
is no shared store to reset between tests and no ordering to preserve.

## A frozen DOM contract

Once a challenge ships in a tagged release as stable, its URL, its
`data-testid` attributes, its behaviour and its manifest entry are fixed.
Courses, blog posts and test suites written against it keep working.

When behaviour genuinely has to differ, the answer is a new page at a new URL
rather than an edit to the old one. [The stability
contract](stability-contract.md) sets out exactly what is promised, what may
still change, and how to check the promise yourself rather than trusting it.

## A control plane that replays

Determinism is only useful if you can turn it off deliberately. Latency,
failure rates, per-challenge flakiness, feature flags and the clock are all
injectable over HTTP, scoped to your session.

Injected chaos is still drawn from your seed, so a test that fails against a
50% failure rate fails the same way on the next run, which is the difference
between a flaky playground and a playground that reproduces flakiness. See [the
control plane](control-plane.md).

## Where to go next

- [Getting started](getting-started.md) — install it, run it, and drive it from
  a suite that runs in parallel.
- [The zone model](zones.md) — why one server hosts several frontends at once,
  and which one to practise in.
- [Challenge catalogue](catalogue.md) — every page, with its selectors,
  endpoints and controls, generated from the manifest.
- [The control plane](control-plane.md) — making your own copy slow, broken or
  flaky.
- [The stability contract](stability-contract.md) — what is frozen, and how to
  detect it moving.
- [Contributing](../CONTRIBUTING.md) — the six rules a new challenge must
  follow.

## What this is not

It is not a mock of any real product, and nothing here is a lesson in how to
build a website. Several pages are deliberately bad: T4 challenges are close to
unautomatable on purpose, and they are shipped so that you can recognise the
shape of them in a codebase, not so that you can copy it.

It is also not a hosted service. There is no instance to depend on, no rate
limit to hit, and no maintainer who can change the page you built a course
around, because the copy you are testing is the one you downloaded.
