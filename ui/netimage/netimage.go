// Package netimage shows pictures from the network, in any window of native UI:
//
//	netimage.Image(c, "https://example.com/cat.jpg").Size(80, 80).Radius(8)
//
// It is what a feed, a chat or a shop needs of every picture: while it comes a
// placeholder holds its place; once it came it fades in. Pictures stay in memory
// as long as there is room, and on disk across runs, as their servers' caching
// headers say, so a list scrolled back shows them at once and a phone offline
// shows what it had. What's on screen downloads before what a fling left behind,
// which doesn't download at all if it's gone by its turn.
package netimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Options say how a picture shows.
type Options struct {
	// Placeholder fills its place until it shows; by default a muted color of
	// the theme.
	Placeholder ui.Color
	// Fit says how it fills what it's given; Cover by default.
	Fit ui.Fit
	// MaxSide bounds the longest side of the bitmap kept, in pixels; by
	// default 2048. Give about what it shows at, on a screen of 3 pixels per
	// DIP: a list of 80 DIP thumbnails holds 240, not a phone's 4000 pixel
	// photos.
	MaxSide int
	// NoFade shows it at once.
	NoFade bool
}

// Image returns an element showing the picture at url: a placeholder, then the
// picture. Give it a size, or it takes the picture's as it comes.
func Image(c *ui.Context, url string, o ...Options) ui.Element {
	var opt Options
	if len(o) > 0 {
		opt = o[0]
	}
	src, age, _ := load(c, url, opt.MaxSide)
	fit := opt.Fit
	if fit == 0 {
		fit = ui.Cover
	}
	if src == nil {
		fill := opt.Placeholder
		if fill == (ui.Color{}) {
			fill = c.Theme().Border
		}
		return ui.Box(c).Background(fill).Role(ui.RoleImage)
	}
	e := ui.Image(c, src).Fit(fit).Role(ui.RoleImage)
	if !opt.NoFade && age >= 0 && age < fadeTime {
		e.Opacity(float32(age) / float32(fadeTime))
		c.AnimationFrame()
	}
	return e
}

// Load returns the picture at url, nil while it comes, to draw it in some other
// way, as an ui.Avatar. The view is built again once it comes. err is what went
// wrong the last time it was asked for: it's asked for again a while later, as
// the view goes on asking.
func Load(c *ui.Context, url string, o ...Options) (src ui.ImageSource, err error) {
	maxSide := 0
	if len(o) > 0 {
		maxSide = o[0].MaxSide
	}
	src, _, err = load(c, url, maxSide)
	return src, err
}

// Preload downloads pictures to disk before they show, as the next page of a
// list that's about to come.
func Preload(urls ...string) {
	for _, u := range urls {
		shared.request(u, defaultSide, nil)
	}
}

// Forget takes a picture out of memory and off the disk: its next showing
// downloads it again.
func Forget(url string) {
	shared.forget(url)
}

// Clear empties the memory and the disk of pictures.
func Clear() {
	shared.clear()
}

// SetClient sets what downloads pictures, as one that signs requests in.
func SetClient(c *http.Client) {
	shared.mu.Lock()
	defer shared.mu.Unlock()
	shared.client = c
}

const (
	fadeTime    = 180 * time.Millisecond
	defaultSide = 2048
	// memoryLimit bounds the pixels held, in bytes.
	memoryLimit = 128 << 20
	// diskLimit bounds the pictures on disk, in bytes.
	diskLimit = 256 << 20
	// workers download at a time.
	workers = 6
	// abandoned is how long after its last showing a picture still waiting
	// is given up: a fling went past it.
	abandoned = 2 * time.Second
	// retryAfter waits before asking again for what failed.
	retryAfter = 10 * time.Second
	// defaultFresh is how long a picture is trusted whose server says nothing.
	defaultFresh = 7 * 24 * time.Hour
)

func load(c *ui.Context, url string, maxSide int) (ui.ImageSource, time.Duration, error) {
	if maxSide <= 0 {
		maxSide = defaultSide
	}
	return shared.request(url, maxSide, c)
}

// key names a picture at a size, in memory.
type key struct {
	url  string
	side int
}

type entry struct {
	key  key
	src  ui.ImageSource
	size int64
	err  error
	// used is its last showing, came when it came to memory, failed when it
	// last failed.
	used, came, failed time.Time
	// loading says a worker has it or it waits for one.
	loading bool
	// fresh says the disk's copy showed at once, not after a wait: no fade.
	fresh bool
	// waiters are the views to build again.
	waiters []*ui.Context
}

type cache struct {
	mu      sync.Mutex
	entries map[key]*entry
	queue   []*entry
	running int
	held    int64
	client  *http.Client
	dir     string
	pruned  bool
	once    sync.Once
	now     func() time.Time
	// limit is memoryLimit, smaller in tests.
	limit int64
}

var shared = newCache()

func newCache() *cache {
	return &cache{entries: map[key]*entry{}, now: time.Now, limit: memoryLimit}
}

// request returns the picture, and starts bringing it if it's not there. c may
// be nil for no view to tell. age is how long ago it came, or -1 for long enough
// not to fade.
func (m *cache) request(url string, side int, c *ui.Context) (ui.ImageSource, time.Duration, error) {
	m.once.Do(func() {
		// Short of memory, the system would rather end the app.
		mygo.App.OnMemoryWarning(m.dropMemory)
	})
	k := key{url, side}
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[k]
	if e == nil {
		e = &entry{key: k}
		m.entries[k] = e
	}
	e.used = now
	if e.src != nil {
		age := now.Sub(e.came)
		if e.fresh {
			age = -1
		}
		return e.src, age, nil
	}
	if c != nil && !slices.Contains(e.waiters, c) {
		e.waiters = append(e.waiters, c)
	}
	if !e.loading && (e.err == nil || now.Sub(e.failed) >= retryAfter) {
		e.loading = true
		m.queue = append(m.queue, e)
		m.startLocked()
	}
	return nil, -1, e.err
}

// startLocked gives workers what waits, the last asked for first: what's on
// screen now.
func (m *cache) startLocked() {
	for m.running < workers && len(m.queue) > 0 {
		e := m.queue[len(m.queue)-1]
		m.queue = m.queue[:len(m.queue)-1]
		if m.now().Sub(e.used) > abandoned {
			e.loading = false
			continue
		}
		m.running++
		go m.bring(e)
	}
}

func (m *cache) bring(e *entry) {
	data, wasFresh, err := m.fetch(e.key.url)
	var src ui.ImageSource
	size := int64(0)
	if err == nil {
		src, size, err = decode(data, e.key.side)
	}
	m.mu.Lock()
	e.loading = false
	e.src, e.err, e.size = src, err, size
	now := m.now()
	if err != nil {
		e.failed = now
	} else {
		e.came, e.fresh = now, wasFresh && now.Sub(e.used) < 50*time.Millisecond
		m.held += size
		m.evictLocked(e)
	}
	waiters := e.waiters
	e.waiters = nil
	m.running--
	m.startLocked()
	m.mu.Unlock()
	for _, c := range waiters {
		c.Invalidate()
	}
}

// decode makes the bitmap: on this goroutine, off the main thread.
func decode(data []byte, side int) (ui.ImageSource, int64, error) {
	b, err := ui.DecodeBitmapScaled(data, side)
	if err != nil {
		// An SVG, which has no pixels to scale.
		s, serr := ui.DecodeImage(data)
		if serr != nil {
			return nil, 0, err
		}
		return s, int64(len(data)), nil
	}
	w, h := b.Size()
	return b, int64(w) * int64(h) * 4, nil
}

// evictLocked lets go of the pictures shown longest ago, down to the limit, but
// keep, which just came.
func (m *cache) evictLocked(keep *entry) {
	if m.held <= m.limit {
		return
	}
	var held []*entry
	for _, e := range m.entries {
		if e.src != nil && e != keep {
			held = append(held, e)
		}
	}
	slices.SortFunc(held, func(a, b *entry) int { return a.used.Compare(b.used) })
	for _, e := range held {
		if m.held <= m.limit {
			return
		}
		m.held -= e.size
		delete(m.entries, e.key)
	}
}

// dropMemory lets go of all the pictures not on screen.
func (m *cache) dropMemory() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	for k, e := range m.entries {
		if e.src != nil && now.Sub(e.used) > time.Second {
			m.held -= e.size
			delete(m.entries, k)
		}
	}
}

func (m *cache) forget(url string) {
	m.mu.Lock()
	for k, e := range m.entries {
		if k.url == url && !e.loading {
			m.held -= e.size
			delete(m.entries, k)
		}
	}
	m.mu.Unlock()
	if body, meta := m.paths(url); body != "" {
		os.Remove(body)
		os.Remove(meta)
	}
}

func (m *cache) clear() {
	m.mu.Lock()
	for k, e := range m.entries {
		if !e.loading {
			m.held -= e.size
			delete(m.entries, k)
		}
	}
	m.mu.Unlock()
	if dir := m.directory(); dir != "" {
		os.RemoveAll(dir)
	}
}

func (m *cache) directory() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return ""
		}
		m.dir = filepath.Join(cache, "mygo-netimage")
	}
	return m.dir
}

func (m *cache) paths(url string) (body, meta string) {
	dir := m.directory()
	if dir == "" {
		return "", ""
	}
	sum := sha256.Sum256([]byte(url))
	name := hex.EncodeToString(sum[:])
	return filepath.Join(dir, name), filepath.Join(dir, name+".json")
}

// metadata is what a picture's response said of its caching.
type metadata struct {
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	FreshUntil   time.Time `json:"freshUntil"`
}

// fetch returns the picture's bytes, from disk while they're fresh, asking the
// server whether they changed once they aren't. wasFresh says it needed no
// network. A phone offline shows what it has, however old.
func (m *cache) fetch(url string) (data []byte, wasFresh bool, err error) {
	body, metaPath := m.paths(url)
	var meta metadata
	var cached []byte
	if body != "" {
		if d, err := os.ReadFile(body); err == nil {
			cached = d
			if raw, err := os.ReadFile(metaPath); err == nil {
				json.Unmarshal(raw, &meta)
			}
			if m.now().Before(meta.FreshUntil) {
				return cached, true, nil
			}
		}
	}
	m.mu.Lock()
	client := m.client
	m.mu.Unlock()
	if client == nil {
		client = http.DefaultClient
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, false, err
	}
	if cached != nil {
		if meta.ETag != "" {
			req.Header.Set("If-None-Match", meta.ETag)
		}
		if meta.LastModified != "" {
			req.Header.Set("If-Modified-Since", meta.LastModified)
		}
	}
	res, err := client.Do(req)
	if err != nil {
		if cached != nil {
			return cached, false, nil
		}
		return nil, false, err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotModified && cached != nil:
		next := m.metadataOf(res)
		if next.ETag == "" {
			next.ETag = meta.ETag
		}
		if next.LastModified == "" {
			next.LastModified = meta.LastModified
		}
		m.writeMeta(metaPath, next)
		return cached, false, nil
	case res.StatusCode != http.StatusOK:
		if cached != nil && res.StatusCode >= 500 {
			return cached, false, nil
		}
		return nil, false, fmt.Errorf("netimage: %s: %s", url, res.Status)
	}
	data, err = io.ReadAll(res.Body)
	if err != nil {
		if cached != nil {
			return cached, false, nil
		}
		return nil, false, err
	}
	if body != "" && !noStore(res) {
		m.store(body, metaPath, data, m.metadataOf(res))
	}
	return data, false, nil
}

func noStore(res *http.Response) bool {
	return strings.Contains(strings.ToLower(res.Header.Get("Cache-Control")), "no-store")
}

// metadataOf reads Cache-Control's max-age and no-cache, then Expires.
func (m *cache) metadataOf(res *http.Response) metadata {
	meta := metadata{ETag: res.Header.Get("ETag"), LastModified: res.Header.Get("Last-Modified")}
	now := m.now()
	fresh := defaultFresh
	found := false
	for part := range strings.SplitSeq(strings.ToLower(res.Header.Get("Cache-Control")), ",") {
		part = strings.TrimSpace(part)
		if part == "no-cache" {
			fresh, found = 0, true
			break
		}
		if v, ok := strings.CutPrefix(part, "max-age="); ok {
			if n, err := strconv.Atoi(v); err == nil {
				fresh, found = time.Duration(n)*time.Second, true
			}
		}
	}
	if !found {
		if t, err := http.ParseTime(res.Header.Get("Expires")); err == nil {
			fresh = t.Sub(now)
		}
	}
	meta.FreshUntil = now.Add(fresh)
	return meta
}

// store writes the picture, whole or not at all, as other goroutines read it.
func (m *cache) store(body, metaPath string, data []byte, meta metadata) {
	dir := filepath.Dir(body)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, "tmp-")
	if err != nil {
		return
	}
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), body)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return
	}
	m.writeMeta(metaPath, meta)
	m.mu.Lock()
	first := !m.pruned
	m.pruned = true
	m.mu.Unlock()
	if first {
		go m.prune(dir, diskLimit)
	}
}

func (m *cache) writeMeta(path string, meta metadata) {
	if raw, err := json.Marshal(meta); err == nil {
		os.WriteFile(path, raw, 0o644)
	}
}

// prune removes what was written longest ago, once a run: a cache may not grow
// without end on a phone.
func (m *cache) prune(dir string, limit int64) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type file struct {
		path string
		size int64
		at   time.Time
	}
	var all []file
	var total int64
	for _, f := range files {
		info, err := f.Info()
		if err != nil || strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		if strings.HasPrefix(f.Name(), "tmp-") {
			os.Remove(filepath.Join(dir, f.Name())) // of a run that ended early
			continue
		}
		all = append(all, file{filepath.Join(dir, f.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	slices.SortFunc(all, func(a, b file) int { return a.at.Compare(b.at) })
	for _, f := range all {
		if total <= limit {
			return
		}
		os.Remove(f.path)
		os.Remove(f.path + ".json")
		total -= f.size
	}
}
