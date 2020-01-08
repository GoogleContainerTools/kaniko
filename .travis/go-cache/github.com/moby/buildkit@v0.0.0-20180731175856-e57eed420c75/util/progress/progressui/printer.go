package progressui

import (
	"fmt"
	"io"
	"time"

	digest "github.com/opencontainers/go-digest"
	"github.com/tonistiigi/units"
)

const antiFlicker = 5 * time.Second
const maxDelay = 10 * time.Second
const minTimeDelta = 5 * time.Second
const minProgressDelta = 0.05 // %

type lastStatus struct {
	Current   int64
	Timestamp time.Time
}

type textMux struct {
	w       io.Writer
	current digest.Digest
	last    map[string]lastStatus
}

func (p *textMux) printVtx(t *trace, dgst digest.Digest) {
	if p.last == nil {
		p.last = make(map[string]lastStatus)
	}

	v, ok := t.byDigest[dgst]
	if !ok {
		return
	}

	if dgst != p.current {
		if p.current != "" {
			old := t.byDigest[p.current]
			if old.logsPartial {
				fmt.Fprintln(p.w, "")
			}
			old.logsOffset = 0
			old.count = 0
			fmt.Fprintf(p.w, "#%d ...\n", v.index)
		}

		fmt.Fprintf(p.w, "\n#%d %s\n", v.index, limitString(v.Name, 72))
	}

	if len(v.events) != 0 {
		v.logsOffset = 0
	}
	for _, ev := range v.events {
		fmt.Fprintf(p.w, "#%d %s\n", v.index, ev)
	}
	v.events = v.events[:0]

	for _, s := range v.statuses {
		if _, ok := v.statusUpdates[s.ID]; ok {
			doPrint := true

			if last, ok := p.last[s.ID]; ok && s.Completed == nil {
				var progressDelta float64
				if s.Total > 0 {
					progressDelta = float64(s.Current-last.Current) / float64(s.Total)
				}
				timeDelta := s.Timestamp.Sub(last.Timestamp)
				if progressDelta < minProgressDelta && timeDelta < minTimeDelta {
					doPrint = false
				}
			}

			if !doPrint {
				continue
			}

			p.last[s.ID] = lastStatus{
				Timestamp: s.Timestamp,
				Current:   s.Current,
			}

			var bytes string
			if s.Total != 0 {
				bytes = fmt.Sprintf(" %.2f / %.2f", units.Bytes(s.Current), units.Bytes(s.Total))
			} else if s.Current != 0 {
				bytes = fmt.Sprintf(" %.2f", units.Bytes(s.Current))
			}
			var tm string
			endTime := s.Timestamp
			if s.Completed != nil {
				endTime = *s.Completed
			}
			if s.Started != nil {
				diff := endTime.Sub(*s.Started).Seconds()
				if diff > 0.01 {
					tm = fmt.Sprintf(" %.1fs", diff)
				}
			}
			if s.Completed != nil {
				tm += " done"
			}
			fmt.Fprintf(p.w, "#%d %s%s%s\n", v.index, s.ID, bytes, tm)
		}
	}
	v.statusUpdates = map[string]struct{}{}

	for i, l := range v.logs {
		if i == 0 {
			l = l[v.logsOffset:]
		}
		fmt.Fprintf(p.w, "%s", []byte(l))
		if i != len(v.logs)-1 || !v.logsPartial {
			fmt.Fprintln(p.w, "")
		}
	}

	if len(v.logs) > 0 {
		if v.logsPartial {
			v.logs = v.logs[len(v.logs)-1:]
			v.logsOffset = len(v.logs[0])
		} else {
			v.logs = nil
			v.logsOffset = 0
		}
	}

	p.current = dgst

	if v.Completed != nil {
		p.current = ""
		v.count = 0
		fmt.Fprintf(p.w, "\n")
	}

	delete(t.updates, dgst)
}

func (p *textMux) print(t *trace) {

	completed := map[digest.Digest]struct{}{}
	rest := map[digest.Digest]struct{}{}

	for dgst := range t.updates {
		v, ok := t.byDigest[dgst]
		if !ok {
			continue
		}
		if v.Vertex.Completed != nil {
			completed[dgst] = struct{}{}
		} else {
			rest[dgst] = struct{}{}
		}
	}

	current := p.current

	// items that have completed need to be printed first
	if _, ok := completed[current]; ok {
		p.printVtx(t, current)
	}

	for dgst := range completed {
		if dgst != current {
			p.printVtx(t, dgst)
		}
	}

	if len(rest) == 0 {
		if current != "" {
			if v := t.byDigest[current]; v.Started != nil && v.Completed == nil {
				return
			}
		}
		// make any open vertex active
		for dgst, v := range t.byDigest {
			if v.Started != nil && v.Completed == nil {
				p.printVtx(t, dgst)
				return
			}
		}
		return
	}

	// now print the active one
	if _, ok := rest[current]; ok {
		p.printVtx(t, current)
	}

	stats := map[digest.Digest]*vtxStat{}
	now := time.Now()
	sum := 0.0
	var max digest.Digest
	if current != "" {
		rest[current] = struct{}{}
	}
	for dgst := range rest {
		v, ok := t.byDigest[dgst]
		if !ok {
			continue
		}
		tm := now.Sub(*v.lastBlockTime)
		speed := float64(v.count) / tm.Seconds()
		overLimit := tm > maxDelay && dgst != current
		stats[dgst] = &vtxStat{blockTime: tm, speed: speed, overLimit: overLimit}
		sum += speed
		if overLimit || max == "" || stats[max].speed < speed {
			max = dgst
		}
	}
	for dgst := range stats {
		stats[dgst].share = stats[dgst].speed / sum
	}

	if _, ok := completed[current]; ok || current == "" {
		p.printVtx(t, max)
		return
	}

	// show items that were hidden
	for dgst := range rest {
		if stats[dgst].overLimit {
			p.printVtx(t, dgst)
			return
		}
	}

	// fair split between vertexes
	if 1.0/(1.0-stats[current].share)*antiFlicker.Seconds() < stats[current].blockTime.Seconds() {
		p.printVtx(t, max)
		return
	}
}

type vtxStat struct {
	blockTime time.Duration
	speed     float64
	share     float64
	overLimit bool
}

func limitString(s string, l int) string {
	if len(s) > l {
		return s[:l] + "..."
	}
	return s
}
