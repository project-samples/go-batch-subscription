package main

import (
	"context"
	"github.com/common-go/config"
	"github.com/common-go/health"

	"go-service/internal/app"
)

func main() {
	var conf app.Root
	er1 := config.Load(&conf, "configs/config")
	if er1 != nil {
		panic(er1)
	}
	ctx := context.Background()

	app, er2 := app.NewApp(ctx, conf)
	if er2 != nil {
		panic(er2)
	}

	go health.Serve(conf.Server, app.HealthHandler)

	app.BatchWorker.RunScheduler(ctx)
	app.Consumer.Consume(ctx, app.ConsumerCaller)
}
