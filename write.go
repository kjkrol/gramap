package main

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/plugins/board/cell"
	"github.com/kjkrol/gram/plugins/board/grid"
	"github.com/kjkrol/gram/plugins/board/water"
)

const attribution = `# Made by gramap. Heights: Terrain Tiles on AWS (Mapzen's Tilezen
# terrarium) — SRTM, EU-DEM, GMTED2010, ETOPO1 and others; see
# https://github.com/tilezen/joerd/blob/master/docs/attribution.md
`

// credits are the comments a map begins with: what it was made from, and of what
func (w *chart) credits(b *bytes.Buffer, recipePath string) {
	b.WriteString(attribution)
	if w.osm != nil {
		b.WriteString("# Roads, tunnels, bridges and buildings: © OpenStreetMap contributors, " +
			"under the ODbL;\n# see https://www.openstreetmap.org/copyright\n")
	}
	if w.covered != nil {
		b.WriteString("# Land cover: CORINE Land Cover 2018, © European Union, Copernicus Land " +
			"Monitoring\n# Service, European Environment Agency; Natura 2000 sites, © EEA\n")
	}
	fmt.Fprintf(b, "# From %s", recipePath)
	if w.flat {
		b.WriteString(", flat")
	}
	b.WriteString(".\n")
	a := w.r.Area
	fmt.Fprintf(b, "# %d x %d cells of %s m", a.Cells[0], a.Cells[1], number(a.Cell))
	if w.cols != a.Cells[0] || w.rows != a.Cells[1] {
		fmt.Fprintf(b, ", %d x %d with the sea", w.cols, w.rows)
	}
	fmt.Fprintf(b, "; a world unit %s m.\n",
		strconv.FormatFloat(math.Round(w.perUnit*1000)/1000, 'f', -1, 64))
	if w.origin != [2]int{} {
		fmt.Fprintf(b, "%s[%d, %d].\n", originLine, w.origin[0], w.origin[1])
	}
}

type standing struct {
	kind   string
	at     cell.ID
	to     cell.ID
	way    int
	bridge bool
}

func (w *chart) standing() []standing {
	var out []standing
	taken := map[cell.ID]bool{}
	for _, s := range w.roads.live() {
		if !s.ford {
			out = append(out, standing{kind: w.r.Kinds.Bridge, at: s.heads[0],
				to: s.heads[len(s.heads)-1], bridge: true})
			continue
		}
		for _, c := range s.cells {
			out = append(out, standing{kind: w.r.Kinds.Ford, at: c, way: s.way})
			taken[c] = true
		}
	}
	for c, course := range w.rivers.Courses {
		if course != water.Ford || taken[c] {
			continue
		}
		down := w.grid.Step(w.roads.wayOf(c, w.rivers.Down[c]))
		way := grid.Nearest(w.grid, geom.NewVec(-down.Y, down.X))
		out = append(out, standing{kind: w.r.Kinds.Ford, at: c, way: way})
	}
	slices.SortFunc(out, func(a, b standing) int { return cmp.Compare(a.at, b.at) })
	return out
}

func (w *chart) deck() float64 {
	if d := w.r.Roads.Deck; d > 0 && d < w.size {
		return d
	}
	return w.size
}

func (w *chart) point(c cell.ID) string {
	x, y := w.xy(c)
	return fmt.Sprintf("[%d, %d]", x, y)
}

// the longest first, so the main stems run whole
func (w *chart) riverPaths() [][]cell.ID {
	n := w.rivers
	fed := map[cell.ID]bool{}
	for c, course := range n.Courses {
		if course != water.Dry {
			fed[n.Down[c]] = true
		}
	}
	length := func(c cell.ID) int {
		l := 0
		for ; n.Courses[c] != water.Dry; c = n.Down[c] {
			l++
		}
		return l
	}
	var sources []cell.ID
	for c, course := range n.Courses {
		if course != water.Dry && !fed[c] {
			sources = append(sources, c)
		}
	}
	slices.SortFunc(sources, func(a, b cell.ID) int {
		return cmp.Or(cmp.Compare(length(b), length(a)), cmp.Compare(a, b))
	})
	taken := map[cell.ID]bool{}
	var out [][]cell.ID
	for _, c := range sources {
		path := []cell.ID{c}
		for taken[c] = true; ; {
			c = n.Down[c]
			path = append(path, c)
			if n.Courses[c] == water.Dry || taken[c] {
				break
			}
			taken[c] = true
		}
		out = append(out, path)
	}
	return out
}

// to the centimetre
func number(v float64) string {
	v = math.Round(v*100) / 100
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func unitsOf(data []byte) string {
	lines := strings.SplitAfter(string(data), "\n")
	for i, l := range lines {
		if !strings.HasPrefix(l, "units:") {
			continue
		}
		end := i + 1
		for end < len(lines) && (lines[end] == "" || strings.HasPrefix(lines[end], " ") ||
			strings.HasPrefix(lines[end], "#") || lines[end] == "\n") {
			end++
		}
		return strings.Join(lines[i:end], "")
	}
	return ""
}

const originLine = "# The recipe's area begins at "

func originOf(data []byte) (x, y int) {
	for l := range strings.Lines(string(data)) {
		if rest, ok := strings.CutPrefix(l, originLine); ok {
			fmt.Sscanf(rest, "[%d, %d].", &x, &y)
			return x, y
		}
	}
	return 0, 0
}

var cellRef = regexp.MustCompile(`\[\s*(-?\d+)\s*,\s*(-?\d+)\s*\]`)

func shiftUnits(units string, dx, dy int) string {
	if dx == 0 && dy == 0 {
		return units
	}
	return cellRef.ReplaceAllStringFunc(units, func(ref string) string {
		m := cellRef.FindStringSubmatch(ref)
		x, _ := strconv.Atoi(m[1])
		y, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("[%d, %d]", x+dx, y+dy)
	})
}
