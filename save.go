package tracetagger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/spacemonkeygo/monkit/v3/collect"
	"github.com/spacemonkeygo/monkit/v3/present"
)

// SaveTracesWithTag saves all traces with the tag into the provided path as a folder
func SaveTracesWithTag(tag interface{}, justTaggedSpans bool, traceMax int, path string) (cancel func()) {
	return TracesWithTag(tag, traceMax, func(spans []*collect.FinishedSpan, capped bool) {
		if justTaggedSpans {
			spans = JustTaggedSpans(spans, tag)
		}
		err := SaveTrace(spans, capped, filepath.Join(path, TracePathPrefix(spans, capped)))
		if err != nil {
			log.Print(err)
		}
	})
}

// JustTaggedSpans filters a list of spans to just the ones that are explicitly
// tagged. It also keeps the first span either way, as that is probably the best
// name for the trace.
func JustTaggedSpans(spans []*collect.FinishedSpan, tag interface{}) (rv []*collect.FinishedSpan) {
	if len(spans) == 0 {
		return spans
	}
	collect.StartTimeSorter(spans).Sort()
	rv = make([]*collect.FinishedSpan, 0, len(spans))
	rv = append(rv, spans[0])
	for _, s := range spans[1:] {
		if IsSpanTagged(s.Span, tag) {
			rv = append(rv, s)
		}
	}
	return rv
}

func sanitize(val string) string {
	return strings.Replace(strings.Map(safechar, val), "..", ".", -1)
}

func safechar(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return r
	}

	switch r {
	case '/':
		return '.'
	case '.', '-':
		return r
	}
	return '_'
}

// TracePathPrefix returns a relative path for the trace with everything but the extension
func TracePathPrefix(spans []*collect.FinishedSpan, capped bool) string {
	if len(spans) == 0 {
		return ""
	}

	collect.StartTimeSorter(spans).Sort()
	traceName := sanitize(spans[0].Span.Func().FullName())

	var end time.Time
	hash := sha256.New()
	for _, s := range spans {
		_, err := hash.Write([]byte(s.Span.Func().FullName() + "\x00"))
		if err != nil {
			panic(err)
		}
		if end.IsZero() || s.Finish.After(end) {
			end = s.Finish
		}
	}
	funcHash := hex.EncodeToString(hash.Sum(nil))

	dir := filepath.Join(traceName, funcHash)
	duration := end.Sub(spans[0].Span.Start())
	filename := filepath.Join(dir, fmt.Sprintf("%d-%d", duration.Nanoseconds(), time.Now().UnixNano()))
	if capped {
		filename += "-capped"
	}

	return filename
}

// SaveTrace saves a trace to pathPrefix with ".json" or ".svg" added.
func SaveTrace(spans []*collect.FinishedSpan, capped bool, pathPrefix string) error {
	if len(spans) == 0 {
		return nil
	}

	err := os.MkdirAll(filepath.Dir(pathPrefix), 0777)
	if err != nil {
		return err
	}

	save := func(saver func(io.Writer, []*collect.FinishedSpan) error, extension string) error {
		fh, err := os.Create(pathPrefix + extension)
		if err != nil {
			return err
		}
		err = saver(fh, spans)
		if err != nil {
			return err
		}
		return fh.Close()
	}

	err = save(present.SpansToJSON, ".json")
	if err != nil {
		return err
	}
	err = save(present.SpansToSVG, ".svg")
	if err != nil {
		return err
	}

	return nil
}
