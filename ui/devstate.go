//go:build !mygo_noinspector

package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/internal/keepstate"
	"github.com/egoist/mygo/internal/surface"
)

// Under mygo dev, a change of the code restarts the app, and Go can't carry a
// running program across: so each window writes down what it can tell of itself,
// as it goes, and the next run takes it back as its elements come. Scroll offsets,
// what a Router shows, and what widgets keep with Local bring the view back as the
// developer left it, whatever the change was; App.KeepState does the same for the
// app's own values. Nothing of this is in release builds.

// devStateEvery is how far behind the app the last words before a restart may be.
const devStateEvery = 300 * time.Millisecond

// devWindows numbers the windows of a run, whose order the restart keeps.
var devWindows atomic.Int32

// keptElement is what an element comes back with.
type keptElement struct {
	X      float64                    `json:"x,omitempty"`
	Y      float64                    `json:"y,omitempty"`
	Locals map[string]json.RawMessage `json:"locals,omitempty"`
}

type keptRouter struct {
	Locs []string `json:"locs"`
	At   int      `json:"at"`
}

// keptWindow is what a window writes down.
type keptWindow struct {
	Elements map[string]*keptElement `json:"elements,omitempty"`
	Routers  map[string]*keptRouter  `json:"routers,omitempty"`
}

func init() {
	devAttach = attachDevState
}

func devStatePath(n int32) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mygo-dev-state", fmt.Sprintf("%s-%d.json", filepath.Base(exe), n)), nil
}

// attachDevState takes back what the last run wrote, and goes on writing it.
func attachDevState(h *windowHost) {
	if !surface.DevLaunched || h.conn.Post == nil {
		return
	}
	path, err := devStatePath(devWindows.Add(1))
	if err != nil {
		return
	}
	rt := h.rt
	rt.dev = &devState{routers: map[uint64]*Router{}}
	var last []byte
	if data, err := os.ReadFile(path); err == nil {
		var k keptWindow
		if json.Unmarshal(data, &k) == nil {
			rt.dev.kept, last = &k, data
		}
	}
	restoreElement, restoreLocal, noteRouter = keptElementFor, keptLocalFor, noteDevRouter
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	go func() {
		for range time.Tick(devStateEvery) {
			done := make(chan []byte, 1)
			h.conn.Post(func() {
				if rt.closed {
					done <- nil
					return
				}
				done <- rt.keptNow()
			})
			data := <-done
			if data == nil {
				if rt.closed {
					return
				}
				continue
			}
			if string(data) == string(last) {
				continue
			}
			last = data
			// Whole or not at all, as the end comes without a word.
			tmp := path + ".tmp"
			if os.WriteFile(tmp, data, 0o600) == nil {
				os.Rename(tmp, path)
			}
		}
	}()
}

// devState is a window's, under mygo dev.
type devState struct {
	// kept is what the last run wrote, until the elements come.
	kept *keptWindow
	// routers are the routers on show, by their element.
	routers map[uint64]*Router
}

func localKey(key any) (string, bool) {
	switch k := key.(type) {
	case string:
		return "s" + k, true
	case int:
		return "i" + strconv.Itoa(k), true
	}
	return "", false
}

// keptNow writes the window down.
func (rt *engine) keptNow() []byte {
	k := keptWindow{Elements: map[string]*keptElement{}}
	for id, s := range rt.states {
		var e keptElement
		e.X, e.Y = s.scrollX, s.scrollY
		for key, v := range s.locals {
			name, ok := localKey(key)
			if !ok {
				continue
			}
			if data := encodeLocal(v); data != nil {
				if e.Locals == nil {
					e.Locals = map[string]json.RawMessage{}
				}
				e.Locals[name] = data
			}
		}
		if e.X != 0 || e.Y != 0 || e.Locals != nil {
			k.Elements[strconv.FormatUint(id, 10)] = &e
		}
	}
	if len(rt.dev.routers) > 0 {
		k.Routers = map[string]*keptRouter{}
		for id, r := range rt.dev.routers {
			kr := &keptRouter{At: r.at}
			for _, e := range r.entries {
				kr.Locs = append(kr.Locs, e.loc)
			}
			k.Routers[strconv.FormatUint(id, 10)] = kr
		}
	}
	data, err := json.Marshal(&k)
	if err != nil {
		return nil
	}
	return data
}

// encodeLocal leaves out what can't be written: a widget's business, not the
// reload's.
func encodeLocal(v any) (data []byte) {
	defer func() {
		if recover() != nil {
			data = nil
		}
	}()
	d, err := keepstate.Encode(v)
	if err != nil || string(d) == "null" || string(d) == "{}" {
		return nil
	}
	return d
}

func keptElementFor(rt *engine, s *state) {
	if rt.dev == nil || rt.dev.kept == nil {
		return
	}
	if e := rt.dev.kept.Elements[strconv.FormatUint(s.id, 10)]; e != nil {
		s.scrollX, s.scrollY = e.X, e.Y
	}
}

func keptLocalFor(s *state, key, v any) {
	rt := s.rt
	if rt == nil || rt.dev == nil || rt.dev.kept == nil {
		return
	}
	name, ok := localKey(key)
	if !ok {
		return
	}
	e := rt.dev.kept.Elements[strconv.FormatUint(s.id, 10)]
	if e == nil || e.Locals[name] == nil {
		return
	}
	defer func() { recover() }() // what no longer fits stays as made
	keepstate.Decode(e.Locals[name], v)
}

// noteDevRouter brings a router back to where it was, the first time it shows.
func noteDevRouter(r *Router, e *node) {
	rt := e.c.rt
	if rt.dev == nil {
		return
	}
	id := e.st.id
	if rt.dev.routers[id] == r {
		return
	}
	rt.dev.routers[id] = r
	if rt.dev.kept == nil {
		return
	}
	if k := rt.dev.kept.Routers[strconv.FormatUint(id, 10)]; k != nil && len(k.Locs) > 0 && k.At >= 0 && k.At < len(k.Locs) {
		r.entries = r.entries[:0]
		for _, loc := range k.Locs {
			r.entries = append(r.entries, &routeEntry{loc: loc})
		}
		r.at = k.At
	}
}
