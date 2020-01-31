package tracetagger

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/spacemonkeygo/monkit/v3/collect"
)

type finishedSpanObserver func(ctx context.Context, s *collect.FinishedSpan)

func (f finishedSpanObserver) Start(ctx context.Context, s *monkit.Span) context.Context { return ctx }

func (f finishedSpanObserver) Finish(ctx context.Context, s *monkit.Span, err error, panicked bool, finish time.Time) {
	f(ctx, &collect.FinishedSpan{Span: s, Err: err, Panicked: panicked, Finish: finish})
}

// TracesWithTag calls the provided observe callback with all spans that
// belong to traces that are no longer running (no current spans with that
// trace exist), where at least one span was tagged with the provided tag.
// It returns a cancel method that will stop new trace collection (existing
// trace collection will keep running)
func TracesWithTag(tag interface{}, traceMax int, observe func(spans []*collect.FinishedSpan, capped bool)) (cancel func()) {
	handle := func(t *monkit.Trace) {
		var mtx sync.Mutex

		var done int32
		var capped bool
		var spans []*collect.FinishedSpan
		var cancel func()

		observer := finishedSpanObserver(func(ctx context.Context, s *collect.FinishedSpan) {
			if atomic.LoadInt32(&done) == 1 {
				return
			}

			active := false
			// this is the worst
			monkit.Default.RootSpans(func(s *monkit.Span) {
				if s.Trace() == t {
					active = true
				}
			})

			pkgTagsMtx.Lock()
			if pkgTags[s.Span.Func().Scope()] == tag {
				tagSpan(s.Span, tag)
			}
			pkgTagsMtx.Unlock()

			var spansToObserve []*collect.FinishedSpan

			mtx.Lock()

			if atomic.LoadInt32(&done) == 1 {
				mtx.Unlock()
				return
			}

			if len(spans) == traceMax {
				capped = true
			} else {
				spans = append(spans, s)
			}

			if !active {
				atomic.StoreInt32(&done, 1)
				cancel()
				spansToObserve = spans
			}
			mtx.Unlock()

			if len(spansToObserve) > 0 {
				if IsTraceTagged(t, tag) {
					observe(spansToObserve, capped)
				}
			}
		})

		mtx.Lock()
		cancel = t.ObserveSpansCtx(observer)
		mtx.Unlock()
	}

	roots := map[*monkit.Trace]bool{}
	monkit.Default.RootSpans(func(s *monkit.Span) {
		if roots[s.Trace()] {
			return
		}
		roots[s.Trace()] = true
		handle(s.Trace())
	})
	cancel = monkit.Default.ObserveTraces(handle)
	roots = nil
	return cancel
}
