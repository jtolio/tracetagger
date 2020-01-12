package main

import (
	"context"
	"log"
	"sync"
	"time"

	monkit "gopkg.in/spacemonkeygo/monkit.v2"
	"gopkg.in/spacemonkeygo/monkit.v2/collect"

	"github.com/jtolds/tracetagger"
)

var (
	mon = monkit.Package()
)

type dbTagT int

var dbTag dbTagT = 1

func DoStep1(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	time.Sleep(100 * time.Millisecond)
	return nil
}

func DoStep2a(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	time.Sleep(200 * time.Millisecond)
	return nil
}

func DoStep2b(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	tracetagger.Tag(ctx, dbTag)
	time.Sleep(300 * time.Millisecond)
	return nil
}

func DoStep2(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	err = DoStep2a(ctx)
	if err != nil {
		return err
	}
	return DoStep2b(ctx)
}

func DoStuff(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	err = DoStep1(ctx)
	if err != nil {
		return err
	}
	return DoStep2(ctx)
}

func main() {
	ctx := context.Background()

	go func() {
		time.Sleep(time.Millisecond)
		err := DoStuff(ctx)
		if err != nil {
			panic(err)
		}
	}()

	var mtx sync.Mutex
	mtx.Lock()

	ccancel := tracetagger.TracesWithTag(dbTag, 1000, func(spans []*collect.FinishedSpan, capped bool) {
		err := tracetagger.SaveTrace(spans, capped, "./traces/")
		if err != nil {
			log.Print(err)
		}
		mtx.Unlock()
	})

	mtx.Lock()
	mtx.Unlock()
	ccancel()
}
