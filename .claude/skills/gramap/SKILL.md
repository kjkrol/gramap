---
name: gramap
description: gramap, the map maker for games on gram — how a map of the Earth is built from a recipe (heights from the Terrain Tiles, rivers drained, roads by gram's roads/site planner and from OpenStreetMap, tunnels, the land cover, fields and settlements, the sea's bed), what decides its yield, its invariants and traps. Use when changing cmd/gramap or a recipe, or making a map anew.
---

# gramap

What the code, its godoc and the README cannot tell. gram's own skills (`gram-plugins`: the board's level
file, `roads/site`, `bridges/site`, `underground/site`, `board/water`) are read from gram's tree.

- Madeira's maps come from `testdata/madeira.yaml`: heights from the Terrain Tiles
  on AWS, rivers by `water.Drain`, roads routed by `roads/site`'s Planner (gramap's `Leaps`
  bridge water and valleys, a leap's cost the bridge's length at the recipe's `bridge` plus its
  heads' earth) and levelled together by `site.Plan` over the whole network after every leg; a
  leg stays only if the network still levels and every bridge stands (`bridges/site.Plan` on the
  graded ground), else the refused cell and its ring are made 3× dearer and it is planned anew,
  48 tries a rung, then left out and named; fords where a river is shallow; the recipe's `roads:`
  are `site.Rules` in metres (`cut`, `fill`, `earth`, `step`, `spread`, `slope`, `fords`, `metre`,
  `cubic`, `climb`) with `route: {cut, fill}` the planner's own allowance (Madeira 40 m against
  200 m of levelling: a planner forgiving the levelling's whole allowance sees no slope between
  cells and runs up gorges the levelling cannot fix); a leg is routed up a ladder of
  rungs, forgiving none of the route allowance first, then a quarter, a half and all of it, and
  the planner sees a cell the levelling could flatten as flat at its corners' mean; a cell
  flanking a diagonal bridge's head is barred (its corners are fixed at the deck and on the
  ground); the map is laid on a board and run a step before it is written, the old file's
  `units:` kept — and `units:` alone: a hand-placed row of `entities:` goes with a regeneration.
  With `roads: {source: osm}` (2026-11-11, a map with relief alone; `-flat` plans the routes
  as before) OpenStreetMap leads the same planner (`osm.go`, `guide.go`, `follow.go`,
  `tunnels.go`): the ways' cells thinned (Zhang–Suen, so two carriageways are one line) and cut
  into legs at junctions; a leg planned in the corridor two cells round its trace, cells off it
  twice dearer, leaps offered only over OpenStreetMap's bridges and water (weighing bridges at
  every expanded node was 89% of the old run); legs levelled sixteen at a time with all laid,
  the one through the refused cell taken out and tried alone, then in halves down to four
  cells, so a long road loses a stretch, never itself. Tunnels first: ways joined into chains,
  a twin tube skipped, the line straightened to the grid's ways (`via`) and led out of each
  head across an edge, two cells on before it slants (`squared`), each head cut down and
  the far corners of its first cell raised (the site wants the ground over the first cell's
  middle at the roof: 2 × clear over the head, 90 m at figures 15), drawn back up to four
  cells; their heads, first cells and corners are pinned for every later levelling (`held`
  plans each anew as `hold` does the bridges). What decides the yield is the 10° of
  `ground.RoadSlope` and the room of figures 15 times life, not the planner. `GRAMAP_MADEIRA=1
  make madeira` holds both of Madeira's maps to their SHA-256. The
  map's one lantern (`{kind: lantern, at: [184, 76]}`) is written by hand: put it back after a
  map is made anew. The sea's bed (`bed.go`, `bed: {zoom, deepest}`, 2026-11-16): the tiles of
  the map's own zoom have the sea at 0, zoom 10 and coarser have its depths (ETOPO1), with
  land told up to 1.5 km past the fine coast: a corner the sea alone touches (`afloat`) takes
  the coarse depth where it is under 0 and is evened between the shore and there where it is
  not (Gauss-Seidel, in a fixed order), whole metres, one at least (a sea cell with a corner
  afloat is always sunk for the terrain), `deepest` at most. Sinking comes last in `build`:
  the rivers drain to a sea at 0.
  Fields and settlements (`fields.go`, `settle.go`, `d/gramap`, 2026-11-06): `fields:` rows
  `{cells, kind, grain, least}` lay a field entity a block of `grain` cells from (0, 0) over the
  cells of a kind where at least `least` lie, written as `{kind, from: [x, y], to: [x, y],
  cells: n}` rows of the board's `entities:` (the demo's `fieldRow`); `settlements:` `{cells,
  sizes, default}` lays `cells` of a kind round each road place, as many as its size's word
  names (a place `[x, y]` or `{at, size}`, `none` for a pass), by a BFS roadside first over land
  that is not wet, road, rock, bridge or another settlement; the legend's `@`. The thresholds,
  the words and the sizes are the game's, in its recipe, sizes in cells never in houses (the
  user's word, 2026-11-06: gramap stays general; what stands on a built cell is the game's
  Unfold rules). `check.go` defines `built` like soil and stand-in field kinds so the map lays
  on a board. A field's `cells:` is a kind or a list of kinds (2026-11-11): one field a block
  over all the land cover's kinds, since a field a kind a block tripled the fields and cost the
  world's publish 0.6 ms of every 30 (Madeira: 315 growth fields and 195 settlements). The land
  cover (`cover.go`, 2026-11-11): CORINE Land Cover 2018's shapes and the Natura 2000 sites
  from the EEA's ArcGIS REST services (GeoJSON, no login, the answers kept in `cover/` beside
  the tiles through the same `overpass` cache), rasterised by the cells' middles a row at a
  time, holes by the even-odd rule; `soil()` takes the cover's kind before the slope's rock
  (the laurisilva grows on slopes: with the rock first it lost 40 % of its cells) and after the
  altitude's. CORINE knows no species: the game's stands pick them by height.
- The map is gram's `board/mapfile` alone (`geography.go`, 2026-11-22; the level file with its
  relief in numbers went with v0.3.0, the user's word): `mapped` makes its files, the folder
  and `<folder>.yaml`, the level over it, `check` lays them on a board, `write` puts them on
  disk. Where this skill speaks of the legend's `@`, of `entities:` rows or of the old file's
  `units:`, read the level over the folder. A flat map is the same without heights; a game's
  world must say its scale, the map being in metres. The heights are 16 bits, 5 cm a step
  over Madeira, and the user's word (2026-11-22) is that the precision stays and the map
  maker copes: `stepFor` picks the step before anything is planned (the bed and the roads'
  fill counted in) and the plan keeps a step clear, the roads' slope a step less a cell (a
  ramp at the 10° limit was 2 cm over it once rounded), the decks' `margin()` a step and a
  half. A bridge is written once, from its span's heads (two roads over one span wrote it
  twice: "a head stands on another bridge"); a ford a road's run, gram keeping one a cell. A
  river's `width` is one a point, its cell's, the sea's cell at a mouth as the last before it
  (a width of zero is refused). An area is a stretch of cells joined by their sides, its
  rings found along the cells' edges with the region on the right; a region touching itself
  by a corner may give one ring passing a point twice, which the even-odd rule reads right.
  The sums in `osm_test.go` are of the files, the heights as the image's values, not the
  PNG's bytes.
