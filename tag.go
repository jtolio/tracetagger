package tracetagger

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/spacemonkeygo/monkit/v3"
)

// TagRef represents a distinct tag reference
type TagRef struct {
	enabled int32
}

// NewTagRef is used for generating new tag references.
// Expected usage like:
//
//  package mypkg
//
//  import (...)
//
//  var DatabaseTag = tracetagger.NewTagRef()
//
func NewTagRef() *TagRef {
	return &TagRef{}
}

// Enable enables the tag for tagging during the run of the function provided
func (t *TagRef) Enable(fn func()) {
	atomic.AddInt32(&t.enabled, 1)
	defer atomic.AddInt32(&t.enabled, -1)
	fn()
}

// Tag attempts to mark the current trace with the provided tag and if
// successful, returns true. It will not tag the current trace if the
// tag is not enabled. Tags are disabled by default.
func Tag(ctx context.Context, tag *TagRef) bool {
	if atomic.LoadInt32(&tag.enabled) > 0 {
		return false
	}
	span := monkit.SpanFromCtx(ctx)
	if span == nil {
		return false
	}
	tagSpan(span, tag)
	return true
}

type spanTag struct {
	tag  *TagRef
	span *monkit.Span
}

func tagSpan(s *monkit.Span, tag *TagRef) {
	s.Trace().Set(tag, true)
	s.Trace().Set(spanTag{tag: tag, span: s}, true)
}

// IsTraceTagged returns true if any span in the trace got tagged.
func IsTraceTagged(t *monkit.Trace, tag *TagRef) bool {
	tagged, ok := t.Get(tag).(bool)
	return ok && tagged
}

// IsSpanTagged returns true if this specific Span got tagged.
func IsSpanTagged(s *monkit.Span, tag *TagRef) bool {
	tagged, ok := s.Trace().Get(spanTag{tag: tag, span: s}).(bool)
	return ok && tagged
}

var pkgTagsMtx sync.Mutex
var pkgTags = map[*monkit.Scope]*TagRef{}

// TagScope tags all functions that belong to the given scope
func TagScope(tag *TagRef, s *monkit.Scope) {
	pkgTagsMtx.Lock()
	pkgTags[s] = tag
	pkgTagsMtx.Unlock()
}
