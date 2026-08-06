# Vendored third-party assets

Checked in rather than fetched, because offline operation is a promise this
project makes. Nothing here is loaded from a CDN, and nothing here has been
modified from upstream.

| Package | Version | Licence | Used by |
|---|---|---|---|
| [jQuery](https://jquery.com) | 3.7.1 | MIT | Zone 2 (`/legacy`) |
| [Bootstrap](https://getbootstrap.com) | 3.4.1 | MIT | Zone 2 (`/legacy`) |

Bootstrap 3 is deliberately old. Zone 2 exists to exercise the jQuery-era
patterns that a large share of real enterprise applications still run on, and
which most modern practice sites have dropped.

## What was left out

- `jquery.slim` and the unminified builds — the minified file carries no
  `sourceMappingURL`, so nothing 404s without them.
- `glyphicons-halflings-regular.eot` and `.svg` — needed only by Internet
  Explorer 8 and pre-5 iOS Safari, neither of which can run the rest of the
  playground. `woff2`, `woff` and `ttf` cover everything that can.
- Bootstrap's JavaScript sourcemap — `bootstrap.min.js` does not reference
  one. `bootstrap.css.map` *is* included, because `bootstrap.min.css` does
  reference it and an unexplained 404 in a tester's devtools is exactly the
  kind of noise this project should not generate.

## Updating

Replace the files from the upstream distribution, unmodified, and update the
version table above. Keep Bootstrap on 3.x: moving Zone 2 to a newer major
would change the DOM those pages produce, and released pages are frozen. A
newer Bootstrap belongs in a new zone.

```sh
npm pack jquery@3 bootstrap@3
```
