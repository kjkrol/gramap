# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this
repository.

## What this is

**gramap** (module `github.com/kjkrol/gramap`) is a command that makes a map of any stretch of
the Earth for a game on [gram](https://github.com/kjkrol/gram): a level file a gram board lays,
from a recipe. It is one package, `main`, at the module's root.

It builds on gram's own rules, never its own copies of them: the roads' planner and levelling
(`plugins/roads/site`), the bridges' (`plugins/bridges/site`), the tunnels'
(`plugins/underground/site`), the draining of a relief (`plugins/board/water`), and it lays
every map on a real world and board before writing it (`check.go`). Locally gram is
`/home/karol/CODE/gram`; this repository is not in `/home/karol/CODE/go.work`, so under
`/home/karol/CODE` it is built with `GOWORK=off`, against the gram its `go.mod` names.

Read `.claude/skills/gramap/SKILL.md` before changing how a map is made; gram's own skills
(`gram-plugins`: the board's level file, `roads/site`, `bridges/site`, `underground/site`) are
read from gram's tree.

## Commands

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./...   # standard verification sequence
go test -run TestName -v .                                       # a single test
make madeira                                                     # Madeira whole, held to its sums
```

## Rules of the house

- **A recipe is a game's.** `testdata/madeira.yaml` is the recipe the tests hold the tool to,
  no game's own: a game keeps its recipe in its repository and may change it freely.
- **A change that should move no cell moves none**: `make madeira` before and after. A change
  meant to move cells says so, and the sums in `osm_test.go` are pinned anew with it.
- **What a board refuses, gramap refuses**: a map is never written past `check`.
- **Nothing is fetched twice**: tiles, OpenStreetMap's answers and the land cover are kept in
  the cache; a test never reaches the network unless `GRAMAP_MADEIRA` is set.
- **Every source is credited** in the map written and in the README.
- **Comments say only what the code cannot**; **lines stay within 100 columns**, a tab counted
  as 4.
- **The user tags and publishes releases**; commit and push only when asked. Commit messages
  use the conventional prefixes and carry no trailer of the tool that wrote them. A new gram
  reaches gramap through a tag of gram and a `go.mod` bump here; `CHANGELOG.md` gets an entry
  per tag.
