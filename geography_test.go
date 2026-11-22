package main

import (
	"math"
	"os"
	"testing"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/plugins/board/mapfile"
	"github.com/kjkrol/gram/plugins/board/water"
)

func geography(t *testing.T, w *chart) *mapfile.File {
	t.Helper()
	dir := t.TempDir()
	if err := w.writeGeography(dir, "isle.yaml"); err != nil {
		t.Fatal(err)
	}
	f, err := mapfile.Load(os.DirFS(dir), manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Whole(); err != nil {
		t.Fatal(err)
	}
	return f
}

func built(t *testing.T) *chart {
	t.Helper()
	r, height := isle(t)
	w, err := build(r, height, nil, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func inside(p geom.Vec, ring []geom.Vec) bool {
	in := false
	for k, a := range ring {
		b := ring[(k+len(ring)-1)%len(ring)]
		if (a.Y > p.Y) != (b.Y > p.Y) && p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
			in = !in
		}
	}
	return in
}

func TestGeography_ItsHeightsAreTheCorners(t *testing.T) {
	w := built(t)
	f := geography(t, w)
	m := float32(w.r.Area.Cell)
	if f.Width != float32(w.cols)*m || f.Height != float32(w.rows)*m {
		t.Fatalf("the map is %g x %g m, want %d x %d cells of %g", f.Width, f.Height, w.cols,
			w.rows, m)
	}
	for y := range w.rows + 1 {
		for x := range w.cols + 1 {
			want := w.corners[y*(w.cols+1)+x]
			got := float64(f.HeightAt(float32(x)*m, float32(y)*m))
			if math.Abs(got-want) > float64(f.Heights.Step)/2+1e-3 {
				t.Fatalf("the corner [%d, %d] is at %g m, want %g", x, y, got, want)
			}
		}
	}
}

func TestGeography_EveryLandCellLiesInTheAreaOfItsKind(t *testing.T) {
	w := built(t)
	// patches of three kinds: holes, a patch within a hole, patches meeting by a corner
	for i := range w.soils {
		x, y := i%w.cols, i/w.cols
		w.soils[i] = []string{"earth", "rock", "sand"}[int(lattice(int64(x/2), int64(y))*3)]
		if dx, dy := x-24, y-12; max(dx, -dx, dy, -dy) == 3 {
			w.soils[i] = "rock"
		}
	}
	f := geography(t, w)
	m := float32(w.r.Area.Cell)
	kinds := map[string]bool{}
	for y := range w.rows {
		for x := range w.cols {
			mid := geom.NewVec((float32(x)+0.5)*m, (float32(y)+0.5)*m)
			var in []string
			for _, a := range f.Features {
				if a.Shape != mapfile.Area || !inside(mid, a.Path) {
					continue
				}
				holed := false
				for _, hole := range a.Holes {
					holed = holed || inside(mid, hole)
				}
				if !holed {
					in = append(in, a.Tags[coverKey])
				}
			}
			switch {
			case !w.land[y*w.cols+x]:
				if len(in) > 0 {
					t.Fatalf("the sea's cell [%d, %d] lies in an area of %v", x, y, in)
				}
			case len(in) != 1 || in[0] != w.kindAt(x, y):
				t.Fatalf("the cell [%d, %d] lies in areas of %v, want %s alone", x, y, in,
					w.kindAt(x, y))
			default:
				kinds[in[0]] = true
			}
		}
	}
	if len(kinds) < 3 {
		t.Fatalf("the island has the kinds %v, want several", kinds)
	}
}

func TestGeography_ItsLinesRunOverTheRiversAndTheRoads(t *testing.T) {
	w := built(t)
	f := geography(t, w)
	m := float32(w.r.Area.Cell)
	type at struct{ x, y int }
	on := map[string]map[at]bool{}
	for _, l := range f.Features {
		if l.Shape != mapfile.Line {
			continue
		}
		key := "waterway"
		if l.Tags["highway"] != "" {
			key = "highway"
			for _, tag := range []string{"bridge", "ford", "tunnel"} {
				if l.Tags[tag] == "yes" {
					key = tag
				}
			}
		}
		if on[key] == nil {
			on[key] = map[at]bool{}
		}
		// a bridge is written from head to head: the cells between are its too
		for k, p := range l.Path {
			on[key][at{int(p.X / m), int(p.Y / m)}] = true
			if k == 0 {
				continue
			}
			q := l.Path[k-1]
			for i := range 64 {
				t := float32(i) / 64
				on[key][at{int((q.X + (p.X-q.X)*t) / m), int((q.Y + (p.Y-q.Y)*t) / m)}] = true
			}
		}
	}
	wet := 0
	for c, course := range w.rivers.Courses {
		if course == water.Dry {
			continue
		}
		wet++
		if x, y := w.xy(c); !on["waterway"][at{x, y}] {
			t.Fatalf("no waterway runs over the wet cell [%d, %d]", x, y)
		}
	}
	if wet == 0 {
		t.Fatal("the island has no river")
	}
	spans := 0
	for _, rd := range w.roads.laid {
		for i, c := range rd.cells {
			x, y := w.xy(c)
			switch {
			case rd.over[i]:
				spans++
				if !on["bridge"][at{x, y}] && !on["ford"][at{x, y}] {
					t.Fatalf("the road's span [%d, %d] is in no line of a bridge or a ford", x, y)
				}
			case !on["highway"][at{x, y}] && !on["bridge"][at{x, y}] && !on["ford"][at{x, y}]:
				t.Fatalf("no highway runs over the road's cell [%d, %d]", x, y)
			}
		}
	}
	if spans == 0 {
		t.Fatal("the island's road spans nothing")
	}
	if s := f.Sections[highway]; float64(s.Lanes)*float64(s.Lane) != w.roads.cfg.Width*w.perUnit {
		t.Fatalf("the road lies %g m across, want %g", float32(s.Lanes)*s.Lane,
			w.roads.cfg.Width*w.perUnit)
	}
	places := map[string]bool{}
	for _, p := range f.Features {
		if p.Shape == mapfile.Point && p.Tags["place"] != "" {
			places[p.Tags["name"]] = true
		}
	}
	if !places["west"] || !places["east"] {
		t.Fatalf("the map names the places %v, want west and east", places)
	}
}

func TestGeography_AFlatMapHasNoHeights(t *testing.T) {
	r, height := isle(t)
	w, err := build(r.asFlat(), height, nil, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if f := geography(t, w); f.Grounded() {
		t.Fatal("a flat map has heights")
	}
}

// laid writes the map in metres with the level over it and lays a board by them
func laid(t *testing.T, w *chart) {
	t.Helper()
	dir := t.TempDir()
	if err := w.writeGeography(dir+"/geography", "isle.yaml"); err != nil {
		t.Fatal(err)
	}
	level := w.levelOver("geography", "isle.yaml", "")
	if err := os.WriteFile(dir+"/map.yaml", level, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.checkIn(os.DirFS(dir), "map.yaml"); err != nil {
		t.Fatal(err)
	}
}

func TestGeography_ABoardLaysTheLevelOverIt(t *testing.T) {
	r, height := isle(t)
	r.stepped = true
	w, err := build(r, height, nil, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if w.step == 0 {
		t.Fatal("a map written in metres is planned in no steps")
	}
	laid(t, w)
}
