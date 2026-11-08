//go:build windows && (amd64 || arm64)

package text

import (
	"fmt"
	"maps"
	"slices"
	"syscall"
	"testing"
)

// TestZZHanProbe logs the fonts Chinese text in no language is drawn in
// with the user's own settings, as an app shows it (#1479).
func TestZZHanProbe(t *testing.T) {
	s := newSystem()
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.engine().(*dwrite)
	for _, text := range []string{"这是一个中文标签，看看字重是否一致。", "我们的音乐播放器正在播放周杰伦的歌曲", "Hello 世界 说话 东西"} {
		r := []rune(text)
		e.resetScratch()
		fams := map[string]bool{}
		for _, line := range e.shapeText(r, Style{Size: 13}, nil, 0, false, false) {
			for _, run := range line.runs {
				if slices.ContainsFunc(r[run.start:run.end], isCJK) {
					name := faceFamily(run.font.native)
					if w, ok := faceWeightAxis(run.font.native); ok {
						name += fmt.Sprintf(" wght %v", w)
					}
					fams[name] = true
				}
			}
		}
		t.Logf("PROBE %q: Han runs in %v", text, slices.Sorted(maps.Keys(fams)))
	}
	t.Logf("PROBE locale %s, Han language %q", syscall.UTF16ToString(e.locale), syscall.UTF16ToString(e.han))
}
