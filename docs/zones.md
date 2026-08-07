# The zone model

The playground is not one application. It is several coexisting frontends under
one server, each built with a different generation of web technique, and each
mounted under its own URL prefix.

<!--generated:zone-table-->

## Why it is built this way

A practice site written entirely in one framework only ever exercises that
framework's happy path. The suites that break in production break because of
what the page is made of: a jQuery widget that replaces its own markup, a React
portal that renders outside its parent, a closed shadow root, a WebSocket that
pushes a change nobody clicked for. A tool that only ever meets one of those
teaches one lesson and hides the rest.

Spanning twenty years of technique on purpose means your framework, your
locator strategy and your waiting strategy are all exercised rather than one
combination of them. It also makes the comparison concrete: the same
underlying problem — a debounced search, a slow response, an element that
leaves mid-interaction — appears in more than one zone, built the way that
generation would have built it, and the automation that survives all of them is
different from the automation that survives one.

The second reason is operational. Zones are independent by construction, so a
zone that cannot build does not take the rest of the product with it. The
Hypermedia zone is declared and not yet populated, and everything else serves
normally regardless.

## What lives where

**Classic** is Go templates with no JavaScript at all. Every interaction is a
form post or a link, so it is where full page loads, redirect chains, status
codes, uploads and downloads, and the whole no-JS fallback story belong. It is
also the zone that punishes holding an element handle across a navigation.

**Legacy** is jQuery 3 and Bootstrap 3, vendored rather than fetched. Partial
updates through `$.ajax`, widgets that replace their own markup, native dialogs
and the older ways of being invisible. It is what most internal enterprise
applications still are.

**Modern SPA** is React 19 with TypeScript, and it is the largest zone. Client
routing, optimistic updates, portals, virtualisation, controlled inputs,
composite flows — the constructs that break assertions rather than locators,
because the DOM says one thing and the server is about to say another.

**Components** is Lit and vanilla custom elements: open and closed shadow
roots, nesting, slots, and events crossing the boundary. It exists to answer
one question honestly, which is what your tooling does when the element is
genuinely unreachable rather than merely awkward.

**Realtime** is vanilla TypeScript over WebSocket and SSE. Updates that arrive
with no triggering action to wait after, connections that drop, streams that
stall rather than fail.

**Hypermedia** is htmx and Alpine over Go templates. It is declared in the zone
list and has no challenges yet; a zone with nothing in it is not advertised in
the manifest.

## What a zone guarantees

A zone is a URL prefix and a technology, and nothing more clever than that.
Specifically:

- **Every challenge lives under its zone's prefix**, and the registry refuses
  to start if one does not. A URL therefore says which technology you are
  about to meet before you open it.
- **Zones share one session.** The seed, the clock and the control-plane rules
  you set apply across all of them, so a latency rule can slow the SPA and the
  Classic zone at once.
- **Zones do not share state.** No challenge in one zone depends on a challenge
  in another having been visited, and none depends on any other challenge at
  all. Any subset of the catalogue is a useful product on its own, which is
  what makes it safe to test one page in isolation.
- **The API for a zone is mounted at `/api/<zone>`**, so the endpoints behind a
  page are as predictable as the page's URL. Each challenge lists its own in
  [the catalogue](catalogue.md).

The one exception to "one server, one origin" is the second port. A browser
decides what is same-origin from scheme, host and port together, so the frame
challenges need a genuinely different socket to embed. That second origin is
the same binary sharing the same session store, described in
[getting started](getting-started.md).

## Choosing where to practise

If you are validating a locator strategy, start in **Components** and
**Modern SPA**: shadow roots and generated class names are where a strategy
that assumed CSS selectors are stable stops working.

If you are validating a waiting strategy, start in **Realtime** and the timing
challenges in **Modern SPA**. A quiet network is not a finished page, and the
pages that teach that are the ones with no request to wait for.

If you are validating a whole framework's basics, or writing training material,
start in **Classic**. It is the zone with the fewest moving parts, so a failure
there is a failure in the tooling rather than an argument about the
application.

If you want to know how your suite behaves when the application is broken
rather than merely difficult, do not change zone. Point [the control
plane](control-plane.md) at the zone you are already in.
