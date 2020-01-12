package main

import (
	"context"
	"time"

	monkit "gopkg.in/spacemonkeygo/monkit.v2"
)

var (
	mon = monkit.Package()
)

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
	err := DoStuff(ctx)
	if err != nil {
		panic(err)
	}
}
