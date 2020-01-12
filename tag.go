package tracetagger

import (
	"context"
	"sync"

	monkit "gopkg.in/spacemonkeygo/monkit.v2"
)

// Tag attempts to mark the current trace with the provided tag and if
// successful, returns true.
func Tag(ctx context.Context, tag interface{}) bool {
	span := monkit.SpanFromCtx(ctx)
	if span == nil {
		return false
	}
	tagSpan(span, tag)
	return true
}

type spanTag struct {
	tag  interface{}
	span *monkit.Span
}

func tagSpan(s *monkit.Span, tag interface{}) {
	s.Trace().Set(tag, true)
	s.Trace().Set(spanTag{tag: tag, span: s}, true)
}

func isTaggedSpan(s *monkit.Span, tag interface{}) bool {
	tagged, ok := s.Trace().Get(spanTag{tag: tag, span: s}).(bool)
	return ok && tagged
}

var pkgTagsMtx sync.Mutex
var pkgTags = map[*monkit.Scope]interface{}{}

// TagScope tags all functions that belong to the given scope
func TagScope(tag interface{}, s *monkit.Scope) {
	pkgTagsMtx.Lock()
	pkgTags[s] = tag
	pkgTagsMtx.Unlock()
}
