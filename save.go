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

	"gopkg.in/spacemonkeygo/monkit.v2/collect"
	"gopkg.in/spacemonkeygo/monkit.v2/present"
)

// SaveTracesWithTag saves all traces with the tag into the provided path as a folder
func SaveTracesWithTag(tag interface{}, justTaggedSpans bool, traceMax int, path string) (cancel func()) {
	return TracesWithTag(tag, justTaggedSpans, traceMax, func(spans []*collect.FinishedSpan, capped bool) {
		err := SaveTrace(spans, capped, path)
		if err != nil {
			log.Print(err)
		}
	})
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

// SaveTrace saves a trace to a folder
func SaveTrace(spans []*collect.FinishedSpan, capped bool, path string) error {
	if len(spans) == 0 {
		return nil
	}

	collect.StartTimeSorter(spans).Sort()
	traceName := sanitize(spans[0].Span.Func().FullName())

	var end time.Time
	hash := sha256.New()
	for _, s := range spans {
		_, err := hash.Write([]byte(s.Span.Func().FullName() + "\x00"))
		if err != nil {
			return err
		}
		if end.IsZero() || s.Finish.After(end) {
			end = s.Finish
		}
	}
	funcHash := hex.EncodeToString(hash.Sum(nil))

	dir := filepath.Join(path, traceName, funcHash)

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	duration := end.Sub(spans[0].Span.Start())

	filename := filepath.Join(dir, fmt.Sprintf("%d-%d", time.Now().UnixNano(), duration.Nanoseconds()))
	if capped {
		filename += "-capped"
	}

	save := func(saver func(io.Writer, []*collect.FinishedSpan) error, extension string) error {
		fh, err := os.Create(filename + extension)
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
