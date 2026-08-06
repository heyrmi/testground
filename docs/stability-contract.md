# The stability contract

Courses, blog posts and training material rot when the site they point at
changes. This project treats that as a defect rather than as the cost of doing
business.

## What is promised

Once a challenge is released in a tagged version with `"stability": "stable"`
in the manifest, the following never change:

- **Its URL.** A link written today resolves to the same page in v1.0 and
  after.
- **Its `data-testid` attributes.** Existing ids keep pointing at the same
  elements. New ones may be added.
- **Its behaviour.** The timings, the outcomes, and which cases succeed or
  fail stay as documented.
- **Its manifest entry.** `id`, `zone`, `tier` and declared selectors are
  fixed. Prose fields may be clarified.
- **Its endpoints.** Request and response shapes stay compatible; fields may
  be added, never removed or repurposed.

## What may change

- **Bug fixes**, where the page did not do what it documented.
- **Visual styling** that leaves the DOM contract intact.
- **New elements, ids and endpoints** alongside the existing ones.
- **Prose** — descriptions and hints get clearer over time.

## When behaviour must change

It does not. A page whose behaviour needs to differ is a **new page at a new
URL**. The old one stays, and its manifest entry gains a pointer to the
replacement.

## Experimental challenges

A challenge marked `"stability": "experimental"` is exempt while it settles.
It says so in the manifest and in a badge on the page itself. Do not build
course material on an experimental challenge; it will be promoted to stable in
a later release, at which point the contract above applies.

## Checking the contract yourself

The manifest is the machine-readable form of this promise, and it can be
diffed:

```sh
playground manifest > manifest.json
git diff manifest.json
```

Committing that file and diffing it in CI is how a course or a test suite
detects that a declared contract moved.

The manifest alone cannot tell you whether the *page* still matches it, so the
reference suite in `examples/playwright-ts` closes that loop: `manifest.spec.ts`
reads every declared selector and looks it up in the live DOM. A `data-testid`
that is renamed in the markup but left alone in the declaration fails there,
which is the drift a manifest diff would miss entirely.

Selectors marked `"transient": true` exist only during an interaction and are
exempt from that presence check. The exemption is itself checked: a transient
selector that turns out to be present on load also fails, so the flag cannot
be used to hide a missing element.

## Determinism is part of the contract

The same seed produces the same content on every run and every machine.
Randomness flows through one seeded source, split into named streams derived
from the seed and the stream name, so a challenge's output never depends on
how many values another challenge happened to draw first. If a page's content
changes for a fixed seed, that is a breaking change and a bug.
