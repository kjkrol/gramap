package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kjkrol/gram/plugins/board/cell"
	"github.com/kjkrol/gram/plugins/board/water"
)

// the files of a map in metres, gram's board/mapfile
const (
	manifestFile = "level.yaml"
	heightsFile  = "heights.png"
	featuresFile = "features.geojson"
)

// the highway a road is written as: OpenStreetMap's word for a road of no class told
const highway = "road"

// the key an area's kind of cell is written under
const coverKey = "landcover"

// drawn is a feature of the map: its id, its tags and its lines, places in cells from the
// north-west corner; a point has one place, a line one line, an area its outline and its holes
type drawn struct {
	id    string
	shape string
	tags  map[string]string
	lines [][][2]float64
}

// writeGeography writes the map into dir as files in metres: the manifest, the heights and
// what lies on the ground
func (w *chart) writeGeography(dir, recipePath string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var b bytes.Buffer
	w.credits(&b, recipePath)
	m := w.r.Area.Cell
	fmt.Fprintf(&b, "size: [%s, %s]\n", metres(float64(w.cols)*m), metres(float64(w.rows)*m))
	if !w.flat {
		img, lowest, step, err := w.heightsImage()
		if err != nil {
			return err
		}
		var data bytes.Buffer
		if err := png.Encode(&data, img); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, heightsFile), data.Bytes(), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(&b, "heights:\n  file: %s\n  pixel: %s\n  lowest: %s\n  step: %s\n",
			heightsFile, metres(m), metres(lowest), metres(step))
	}
	fmt.Fprintf(&b, "features: %s\n", featuresFile)
	if len(w.roads.laid) > 0 {
		fmt.Fprintf(&b, "roads:\n  %s:\n    lanes: 2\n    lane: %s\n", highway,
			metres(w.roads.cfg.Width*w.perUnit/2))
	}
	if err := os.WriteFile(filepath.Join(dir, manifestFile), b.Bytes(), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, featuresFile), w.geoJSON(w.features()), 0o644)
}

// writeLevels writes the map into the folder dir as files in metres and, beside it and named
// as it, the level a board lays by it with the game's units, laid on a board first without
// them: their kinds are the game's
func (w *chart) writeLevels(dir, recipePath, units string) error {
	if err := w.writeGeography(dir, recipePath); err != nil {
		return err
	}
	folder, level := filepath.Base(dir), filepath.Clean(dir)+".yaml"
	for _, units := range []string{"", units} {
		data := w.levelOver(folder, recipePath, units)
		if err := os.WriteFile(level, data, 0o644); err != nil {
			return err
		}
		if units != "" {
			break
		}
		if err := w.checkIn(os.DirFS(filepath.Dir(level)), filepath.Base(level)); err != nil {
			return err
		}
	}
	return nil
}

// levelOver is the level a board lays by the map in the folder beside it: the kinds its
// features are laid as, the fields and the units
func (w *chart) levelOver(folder, recipePath, units string) []byte {
	var b bytes.Buffer
	w.credits(&b, recipePath)
	k := w.r.Kinds
	fmt.Fprintf(&b, "board:\n  default: %s\n", k.Sea)
	if fields := w.fields(); len(fields) > 0 {
		b.WriteString("  entities:\n")
		for _, f := range fields {
			fmt.Fprintf(&b, "    - {kind: %s, from: %s, to: %s, cells: %d}\n", f.kind,
				w.point(f.from), w.point(f.to), f.cells)
		}
	}
	fmt.Fprintf(&b, "geography:\n  file: %s/%s\n", folder, manifestFile)
	if len(w.roads.laid) > 0 {
		fmt.Fprintf(&b, "  roads: {%s: %s}\n  bridges: {%s: %s}\n", highway, k.Road, highway,
			k.Bridge)
		if len(w.roads.tunnels) > 0 {
			fmt.Fprintf(&b, "  tunnels: {%s: %s}\n", highway, k.Tunnel)
		}
	}
	fmt.Fprintf(&b, "  fords: %s\n", k.Ford)
	// a course's width is one a kind here: the mean of its cells'
	wide, cells := map[water.Course]float64{}, map[water.Course]float64{}
	w.grid.EachCell(func(c cell.ID) {
		course := w.rivers.Courses[c]
		if course == water.Ford {
			course = water.River
		}
		wide[course] += float64(w.rivers.Width(c, float32(w.size)))
		cells[course]++
	})
	var rivers []string
	for _, r := range []struct {
		course water.Course
		name   string
		kind   string
	}{{water.Brook, "brook", k.Brook}, {water.Stream, "stream", k.Stream},
		{water.River, "river", k.River}} {
		if r.course != water.Dry && cells[r.course] > 0 {
			rivers = append(rivers, fmt.Sprintf("    %s: {kind: %s, width: %s}\n", r.name, r.kind,
				number(wide[r.course]/cells[r.course])))
		}
	}
	if len(rivers) > 0 {
		b.WriteString("  rivers:\n" + strings.Join(rivers, ""))
	}
	var kinds []string
	for i, kind := range w.soils {
		if w.land[i] && !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}
	slices.Sort(kinds)
	if len(kinds) > 0 {
		b.WriteString("  areas:\n")
		for _, kind := range kinds {
			fmt.Fprintf(&b, "    %s=%s: %s\n", coverKey, kind, kind)
		}
	}
	b.WriteString(units)
	return b.Bytes()
}

func metres(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// the steps of a height's value a map may have, metres: the finest that reaches from its
// lowest corner to its highest serves
var heightSteps = [...]float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1}

// stepFor is the step the heights are written in: the finest whose values reach from the
// sea's bed to the highest ground with what a road may raise it
func (w *chart) stepFor() float64 {
	lowest, highest := slices.Min(w.corners), slices.Max(w.corners)+w.r.Roads.Fill
	if w.r.Bed.Zoom > 0 {
		lowest -= w.r.Bed.Deepest
	}
	for _, s := range heightSteps {
		if (highest-lowest)/s <= math.MaxUint16 {
			return s
		}
	}
	return heightSteps[len(heightSteps)-1]
}

// heightsImage is the corners' heights as an image, a pixel a corner, with the height of the
// value 0 and the metres a step of the value
func (w *chart) heightsImage() (img *image.Gray16, lowest, step float64, err error) {
	lowest, step = math.Floor(slices.Min(w.corners)), w.step
	if step == 0 {
		step = w.stepFor()
	}
	if top := slices.Max(w.corners); (top-lowest)/step > math.MaxUint16 {
		return nil, 0, 0, fmt.Errorf("the ground reaches from %g to %g m: more than steps of "+
			"%g m tell", lowest, top, step)
	}
	img = image.NewGray16(image.Rect(0, 0, w.cols+1, w.rows+1))
	for y := range w.rows + 1 {
		for x := range w.cols + 1 {
			v := math.Round((w.corners[y*(w.cols+1)+x] - lowest) / step)
			img.SetGray16(x, y, color.Gray16{Y: uint16(v)})
		}
	}
	return img, lowest, step, nil
}

// features is what lies on the map: the kinds of its land as areas, its rivers and roads as
// lines, its fords and its roads' places as points
func (w *chart) features() []drawn {
	out := w.areas()
	out = append(out, w.waterways()...)
	out = append(out, w.highways()...)
	// a river's own fords; a road's are its line's
	for _, s := range w.standing() {
		if !s.bridge && w.roads.spanOf[s.at] == nil {
			out = append(out, drawn{shape: "Point", tags: map[string]string{"ford": "yes"},
				lines: [][][2]float64{{w.middle(s.at)}}})
		}
	}
	for _, name := range slices.Sorted(maps.Keys(w.roads.places)) {
		tags := map[string]string{"name": name, "place": "locality"}
		if size := w.r.Roads.Places[name].Size; size != "" {
			tags["place"] = size
		}
		out = append(out, drawn{shape: "Point", tags: tags,
			lines: [][][2]float64{{w.middle(w.roads.places[name])}}})
	}
	counts := map[string]int{}
	for i := range out {
		lead := out[i].shape
		for _, key := range []string{coverKey, "waterway", "highway", "ford", "place"} {
			if _, ok := out[i].tags[key]; ok {
				lead = key
				break
			}
		}
		counts[lead]++
		out[i].id = lead + "/" + strconv.Itoa(counts[lead])
	}
	return out
}

func (w *chart) middle(c cell.ID) [2]float64 {
	x, y := w.xy(c)
	return [2]float64{float64(x) + 0.5, float64(y) + 0.5}
}

func (w *chart) middles(cells []cell.ID) [][2]float64 {
	out := make([][2]float64, len(cells))
	for i, c := range cells {
		out[i] = w.middle(c)
	}
	return out
}

// waterways are the rivers, a line a stretch of one course: its last place where the water
// goes, its width the mean of its cells'
func (w *chart) waterways() []drawn {
	names := map[water.Course]string{
		water.Brook: "brook", water.Stream: "stream", water.River: "river", water.Ford: "river",
	}
	var out []drawn
	for _, path := range w.riverPaths() {
		for from := 0; from < len(path)-1; {
			name, wide := names[w.rivers.Courses[path[from]]], 0.0
			to := from
			for ; to < len(path)-1 && names[w.rivers.Courses[path[to]]] == name; to++ {
				wide += float64(w.rivers.Width(path[to], float32(w.size)))
			}
			wide = wide / float64(to-from) * w.perUnit
			out = append(out, drawn{
				shape: "LineString",
				tags: map[string]string{
					"waterway": name,
					"width":    metres(math.Round(wide*100) / 100),
				},
				lines: [][][2]float64{w.middles(path[from : to+1])},
			})
			from = to
		}
	}
	return out
}

// highways are the roads, a line a stretch on the ground, a bridge from head to head, a ford
// or a tunnel from the cell before it to the cell after
func (w *chart) highways() []drawn {
	var out []drawn
	add := func(cells []cell.ID, tag string) {
		tags := map[string]string{"highway": highway}
		if tag != "" {
			tags[tag] = "yes"
		}
		out = append(out, drawn{shape: "LineString", tags: tags,
			lines: [][][2]float64{w.middles(cells)}})
	}
	over := func(rd road, i int) string {
		switch {
		case !rd.over[i]:
			return ""
		case w.roads.spanOf[rd.cells[i]] != nil && w.roads.spanOf[rd.cells[i]].ford:
			return "ford"
		}
		return "bridge"
	}
	// a bridge is its span's, from head to head, once whatever roads run over it
	bridged := map[*span]bool{}
	for _, rd := range w.roads.laid {
		for from := 0; from < len(rd.cells); {
			tag, to := over(rd, from), from
			for to+1 < len(rd.cells) && over(rd, to+1) == tag {
				to++
			}
			switch s := w.roads.spanOf[rd.cells[from]]; {
			case tag == "" && to > from:
				add(rd.cells[from:to+1], "")
			case tag == "bridge" && !bridged[s]:
				bridged[s] = true
				add([]cell.ID{s.heads[0], s.heads[len(s.heads)-1]}, tag)
			case tag == "ford":
				// from the cell before the ford to the cell after
				add(rd.cells[max(from-1, 0):min(to+1, len(rd.cells)-1)+1], tag)
			}
			from = to + 1
		}
	}
	for _, t := range w.roads.tunnels {
		cells := append([]cell.ID{t.from}, t.plan.Cells...)
		add(append(cells, t.to), "tunnel")
	}
	return out
}

// areas are the land's kinds, an area a stretch of cells of one kind joined by their sides:
// its outline along the cells' edges, the stretches of other kinds within it its holes
func (w *chart) areas() []drawn {
	region := make([]int, w.cols*w.rows)
	var kinds []string
	for i := range region {
		x, y := i%w.cols, i/w.cols
		if region[i] != 0 || !w.land[i] {
			continue
		}
		kinds = append(kinds, w.kindAt(x, y))
		region[i] = len(kinds)
		for queue := []int{i}; len(queue) > 0; queue = queue[1:] {
			cx, cy := queue[0]%w.cols, queue[0]/w.cols
			for _, d := range sides {
				nx, ny := cx+d[0], cy+d[1]
				if nx < 0 || ny < 0 || nx >= w.cols || ny >= w.rows {
					continue
				}
				n := ny*w.cols + nx
				if region[n] == 0 && w.land[n] && w.kindAt(nx, ny) == kinds[len(kinds)-1] {
					region[n] = len(kinds)
					queue = append(queue, n)
				}
			}
		}
	}
	at := func(x, y int) int {
		if x < 0 || y < 0 || x >= w.cols || y >= w.rows {
			return 0
		}
		return region[y*w.cols+x]
	}
	// a region's edges by the corner each leaves, the region on the right going along
	edges := make([]map[int][]int, len(kinds)+1)
	stride := w.cols + 1
	for y := range w.rows {
		for x := range w.cols {
			r := region[y*w.cols+x]
			if r == 0 {
				continue
			}
			k := y*stride + x
			corners := [4]int{k, k + 1, k + stride + 1, k + stride}
			for way, d := range sides {
				if at(x+d[0], y+d[1]) == r {
					continue
				}
				if edges[r] == nil {
					edges[r] = map[int][]int{}
				}
				edges[r][corners[way]] = append(edges[r][corners[way]], way)
			}
		}
	}
	rings := make([][][][2]float64, len(kinds)+1)
	step := [4]int{1, stride, -1, -stride}
	take := func(r, from, way int) bool {
		ways := edges[r][from]
		i := slices.Index(ways, way)
		if i < 0 {
			return false
		}
		edges[r][from] = slices.Delete(ways, i, i+1)
		return true
	}
	for y := range w.rows {
		for x := range w.cols {
			r := region[y*w.cols+x]
			start := y*stride + x
			if r == 0 || !take(r, start, 0) {
				continue
			}
			var ring [][2]float64
			for at, way := start, 0; ; {
				at += step[way]
				next := -1
				// the turn to the right first: a region touching itself by a corner stays one ring
				for _, turn := range [3]int{1, 0, 3} {
					if take(r, at, (way+turn)%4) {
						next = (way + turn) % 4
						break
					}
				}
				if next != way {
					ring = append(ring, [2]float64{float64(at % stride), float64(at / stride)})
				}
				if next < 0 {
					break
				}
				way = next
			}
			// the ring found first, from the region's first cell, is its outline
			rings[r] = append(rings[r], ring)
		}
	}
	var out []drawn
	for r := 1; r <= len(kinds); r++ {
		out = append(out, drawn{shape: "Polygon",
			tags: map[string]string{coverKey: kinds[r-1]}, lines: rings[r]})
	}
	return out
}

// north, east, south, west: a cell's sides in the order its edges run, the cell on the right
var sides = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// geoJSON is the features as a GeoJSON collection, metres east and north of the map's
// south-west corner, to the centimetre
func (w *chart) geoJSON(features []drawn) []byte {
	m := w.r.Area.Cell
	place := func(p [2]float64) [2]float64 {
		return [2]float64{
			math.Round(p[0]*m*100) / 100,
			math.Round((float64(w.rows)-p[1])*m*100) / 100,
		}
	}
	type geometry struct {
		Type        string `json:"type"`
		Coordinates any    `json:"coordinates"`
	}
	type row struct {
		Type       string            `json:"type"`
		ID         string            `json:"id"`
		Properties map[string]string `json:"properties"`
		Geometry   geometry          `json:"geometry"`
	}
	var b bytes.Buffer
	b.WriteString("{\"type\": \"FeatureCollection\", \"features\": [\n")
	for i, f := range features {
		lines := make([][][2]float64, len(f.lines))
		for k, line := range f.lines {
			for _, p := range line {
				lines[k] = append(lines[k], place(p))
			}
			if f.shape == "Polygon" {
				lines[k] = append(lines[k], lines[k][0])
			}
		}
		var coordinates any = lines
		switch f.shape {
		case "Point":
			coordinates = lines[0][0]
		case "LineString":
			coordinates = lines[0]
		}
		data, _ := json.Marshal(row{"Feature", f.id, f.tags, geometry{f.shape, coordinates}})
		b.Write(data)
		if i < len(features)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("]}\n")
	return b.Bytes()
}
