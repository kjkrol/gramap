# kjkrol/gramap

<p align="center">
  <a href="https://go.dev">
    <img src="https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  </a>
  <a href="https://pkg.go.dev/github.com/kjkrol/gramap">
    <img src="https://img.shields.io/badge/GoDoc-Reference-007d9c?style=flat-square&logo=go" alt="GoDoc">
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square" alt="License">
  </a>
  <a href="https://github.com/kjkrol/gramap/actions">
    <img src="https://github.com/kjkrol/gramap/actions/workflows/go.yml/badge.svg" alt="Go Quality Check">
  </a>
</p>

**gramap** makes a map of any stretch of the Earth, at any scale, for a game on
[gram](https://github.com/kjkrol/gram): a map in files, in metres, with its relief, its kinds
of ground, its rivers, its roads, bridges, fords and tunnels, and the level a gram board lays
by it, written from a recipe that says where and how. The map is laid on a real board before
it is written, so what a board refuses, gramap refuses.

## Use

gramap requires **Go 1.27** or newer.

```bash
go run github.com/kjkrol/gramap@latest -recipe madeira.yaml -geography relief/geography
go run github.com/kjkrol/gramap@latest -recipe madeira.yaml -geography flat/geography -flat
```

| Flag | What it is |
|---|---|
| `-recipe` | the recipe, YAML |
| `-geography` | the folder to write the map into; `<folder>.yaml`, the level over it, is written beside it, a `units:` section in the one made before staying as it was |
| `-flat` | a map without heights, its bridges a cell long |
| `-cache` | where what is fetched is kept; unset, `gramap` in the user's cache |

## The map

The map is written as gram's
[`board/mapfile`](https://github.com/kjkrol/gram/tree/main/plugins/board/mapfile) has one, a
folder: `level.yaml`, the ground's heights as `heights.png`, a pixel a corner of a cell (none
with `-flat`), and `features.geojson`, in metres, named as OpenStreetMap's tags are:

| What | Written as |
|---|---|
| the land's kinds | areas, `landcover=<kind>`, along the cells' edges |
| the rivers | lines, `waterway=brook`, `stream` or `river`, its `width` a point |
| the roads | lines, `highway=road` |
| the bridges | lines from head to head, `highway=road` and `bridge=yes` |
| the roads' fords and the tunnels | lines from the cell before to the cell after, `ford=yes`, `tunnel=yes` |
| the rivers' fords | points, `ford=yes` |
| the roads' places | points, `name` and `place` (its size) |

Beside the folder lies `<folder>.yaml`, the level a gram board lays by it: the kinds the
map's features are laid as, the fields and the game's units. A game's world says how many
metres its unit is (`world.Config.Scale`), as the map's first lines do.

The heights are 16 bits a pixel, so over a ground reaching far up they are told in steps of
centimetres (5 cm over Madeira). A map with relief is planned a step clear of every limit, its
roads climbing a step less a cell and its decks standing a step and a half further off the
ground, so no rounding lays a road or a bridge the board refuses.

## The recipe

A recipe says where, how many cells and how many metres a cell spans, what the ground is by
its slope and height, how much drained land makes a brook, a stream and a river, and the roads
as chains of named places. `testdata/madeira.yaml` is a whole one: Madeira, 640 by 320 cells of
100 m.

- **Heights** come from the [Terrain Tiles on AWS](https://registry.opendata.aws/terrain-tiles/),
  fetched once into the cache. With `bed: {zoom, deepest}` the sea has its bed, as deep as the
  tiles tell at a zoom that has the sea's depths, evened from the shore to there.
- **Rivers** are carved into the ground where enough land drains.
- **Roads** are routed from place to place and graded as Transport Tycoon lays them, by gram's
  own road rules (`plugins/roads/site`), with bridges over water and down valleys and fords
  where a river is shallow. With `roads: {source: osm}` a map with relief follows
  [OpenStreetMap](https://www.openstreetmap.org/copyright) instead: its roads of the classes
  the recipe names, its bridges, its tunnels dug where the ground lets one lie, and its
  settlements where the buildings stand.
- **Land cover**: with `cover: {source: corine}` the land is drawn as
  [CORINE Land Cover](https://land.copernicus.eu/en/products/corine-land-cover) says, a kind of
  cell for the classes the recipe names, another within a Natura 2000 site.
- **Fields and settlements**: `fields:` lay a field entity over the cells of a kind in each
  block where enough of them lie, `settlements:` cells of a kind round the roads' places, as
  many as a place's size says. What grows and stands on them is the game's rules'.

The command's godoc (`main.go`) has every section in full.

## Testing

```bash
make build test
make madeira        # Madeira made whole and held to its sums
```

`make madeira` makes both of Madeira's maps from what the cache holds (two minutes; the first
run fetches), lays each on a board and compares its files with their SHA-256: a change that should move no cell moves none.

## Credits

A map gramap makes carries what it was made from:

- Heights: the Terrain Tiles on AWS, Mapzen's Tilezen terrarium (SRTM, EU-DEM (Copernicus),
  GMTED2010, ETOPO1 and others); see their
  [attribution](https://github.com/tilezen/joerd/blob/master/docs/attribution.md).
- With `source: osm`: © [OpenStreetMap](https://www.openstreetmap.org/copyright) contributors,
  under the Open Database License.
- With `cover: {source: corine}`: CORINE Land Cover 2018, © European Union, Copernicus Land
  Monitoring Service, European Environment Agency (EEA); the Natura 2000 sites, © EEA.

## License

MIT, see [LICENSE](LICENSE).
