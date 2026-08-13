package main

import (
	"context"
	"log"

	"github.com/dollhouse-app/dollhouse/backend/internal/config"
	"github.com/dollhouse-app/dollhouse/backend/internal/handler"
	"github.com/dollhouse-app/dollhouse/backend/internal/profiledoll"
)

func main() {
	handlers, err := profiledoll.NewRuntimeHandlers(context.Background(), config.Load())
	if err != nil {
		log.Fatal(err)
	}
	handler.StartAPIHandler(handlers.UpdateDoll)
}
