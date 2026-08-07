# Challenge catalogue

Everything on this page is generated from `playground manifest`, which is the
same document the running server publishes at `/api/challenges`. Nothing here
is typed by hand, because a hand-maintained list of the same facts is a second
copy that drifts the first time somebody adds a page and updates only one of
them. If a challenge is missing below, it is missing from the binary.

<!--generated:catalogue-stats-->

Difficulty runs T1 for an introduction to T4 for pages that are deliberately
close to unautomatable. T4 exists to be recognised rather than solved: it is
shipped so that you can name the shape of it when you meet it in a real
codebase.

Prefer the JSON to this page whenever a machine is reading. Page-object
generators, coverage tooling and contract diffs should all consume the
manifest directly:

```sh
playground manifest > manifest.json          # no server needed
curl -s localhost:7373/api/challenges        # or from a running one
curl -s localhost:7373/api/challenges/virtual-list
```

Committing that file and diffing it in CI is how a suite detects a contract
moving; see [the stability contract](stability-contract.md) for what is
promised not to.

## Every challenge

<!--generated:challenge-index-->

<!--generated:catalogue-->
