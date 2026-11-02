package keepstate

import (
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
)

type inner struct {
	Name  string
	count int
}

type thing struct {
	tab     int
	name    string
	on      bool
	ratio   float64
	big     int64
	rows    []string
	byName  map[string]int
	byID    map[int]inner
	in      inner
	when    time.Time
	took    time.Duration
	next    *inner
	empty   []int
	arr     [2]int
	mu      sync.Mutex
	fn      func()
	ch      chan int
	foreign *scene.Image
	any     any
	nan     float64
}

// A struct of an app's comes back whole, unexported fields and all.
func TestRoundTrip(t *testing.T) {
	when := time.Date(2026, 11, 2, 12, 0, 0, 5, time.UTC)
	a := &thing{
		tab: 3, name: "中文 😀", on: true, ratio: 0.25, big: math.MaxInt64 - 1,
		rows: []string{"a", "b"}, byName: map[string]int{"x": 1}, byID: map[int]inner{7: {"seven", 7}},
		in: inner{"in", 2}, when: when, took: time.Minute, next: &inner{"next", 9}, arr: [2]int{4, 5},
		nan: math.NaN(),
	}
	data, err := Encode(a)
	if err != nil {
		t.Fatal(err)
	}
	var b thing
	if err := Decode(data, &b); err != nil {
		t.Fatal(err)
	}
	if b.tab != 3 || b.name != a.name || !b.on || b.ratio != 0.25 || b.big != a.big {
		t.Errorf("scalars: %d %q %v", b.tab, b.name, b.ratio)
	}
	if !reflect.DeepEqual(b.rows, a.rows) || !reflect.DeepEqual(b.byName, a.byName) || !reflect.DeepEqual(b.byID, a.byID) {
		t.Errorf("collections: %v %v %v", b.rows, b.byName, b.byID)
	}
	if b.in != a.in || !b.when.Equal(when) || b.took != time.Minute || b.next == nil || *b.next != *a.next || b.arr != a.arr {
		t.Errorf("structs: %+v, %v", b.in, b.next)
	}
}

// What it doesn't keep stays as the program made it.
func TestLeavesTheWorld(t *testing.T) {
	made := func() {}
	ch := make(chan int)
	img := &scene.Image{W: 1}
	a := &thing{tab: 1, foreign: &scene.Image{W: 9}, any: 5}
	data, err := Encode(a)
	if err != nil {
		t.Fatal(err)
	}
	b := thing{fn: made, ch: ch, foreign: img, any: "made", ratio: 2}
	if err := Decode(data, &b); err != nil {
		t.Fatal(err)
	}
	if b.fn == nil || b.ch != ch || b.foreign != img || img.W != 1 || b.any != "made" {
		t.Errorf("the world changed: %v %v", b.foreign, b.any)
	}
	if b.tab != 1 {
		t.Errorf("tab: %d", b.tab)
	}
}

// A change of the code keeps what still fits.
func TestCodeChanged(t *testing.T) {
	type before struct {
		tab   string
		kept  int
		gone  int
		rows  []int
		byKey map[string]int
	}
	type after struct {
		tab   int // another type: stays as made
		kept  int
		added string
		rows  []string // another type of element: its elements stay empty
		byKey map[int]int
	}
	data, err := Encode(&before{tab: "x", kept: 5, gone: 9, rows: []int{1}, byKey: map[string]int{"a": 1}})
	if err != nil {
		t.Fatal(err)
	}
	b := after{tab: 7, added: "made"}
	if err := Decode(data, &b); err != nil {
		t.Fatal(err)
	}
	if b.tab != 7 || b.kept != 5 || b.added != "made" {
		t.Errorf("%+v", b)
	}
}

// A cycle ends.
func TestCycle(t *testing.T) {
	type node struct {
		next *node
		n    int
	}
	a := &node{n: 1}
	a.next = a
	data, err := Encode(a)
	if err != nil {
		t.Fatal(err)
	}
	var b node
	if err := Decode(data, &b); err != nil {
		t.Fatal(err)
	}
	if b.n != 1 {
		t.Errorf("n: %d", b.n)
	}
}

// Numbers that overflow what they go into stay out.
func TestOverflow(t *testing.T) {
	type big struct{ n int64 }
	type small struct{ n int8 }
	data, err := Encode(&big{n: 1000})
	if err != nil {
		t.Fatal(err)
	}
	b := small{n: 1}
	if err := Decode(data, &b); err != nil {
		t.Fatal(err)
	}
	if b.n != 1 {
		t.Errorf("n: %d", b.n)
	}
}

// Encoding a plain scalar, as a widget's local state is.
func TestScalar(t *testing.T) {
	n := 42
	data, err := Encode(&n)
	if err != nil {
		t.Fatal(err)
	}
	var m int
	if err := Decode(data, &m); err != nil || m != 42 {
		t.Errorf("%d, %v", m, err)
	}
}
