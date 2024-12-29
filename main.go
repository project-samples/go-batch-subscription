package main

import (
	"context"

	"github.com/core-go/config"
	"github.com/core-go/health/server"

	"go-service/internal/app"
)

func main() {
	var cfg app.Config
	er1 := config.Load(&cfg, "configs/config")
	if er1 != nil {
		panic(er1)
	}
	ctx := context.Background()

	app, er2 := app.NewApp(ctx, cfg)
	if er2 != nil {
		panic(er2)
	}

	go server.Serve(cfg.Server, app.Health.Check)
	app.Run(ctx)
	app.Receive(ctx, app.Handle)
}
