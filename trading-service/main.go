package main

import (
	"context"
	"github.com/lukasz-zimnoch/dexly/trading-service/pkg/job"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	job.RunTrading(ctx)

	<-ctx.Done()
}
