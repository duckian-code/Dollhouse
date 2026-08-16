package main

import (
	"context"
	"log"

	"github.com/dollhouse-app/dollhouse/backend/internal/config"
	"github.com/dollhouse-app/dollhouse/backend/internal/friendship"
	"github.com/dollhouse-app/dollhouse/backend/internal/handler"
)

func main() {
	handlers, err := friendship.NewRuntimeHandlers(context.Background(), config.Load())
	if err != nil {
		log.Fatal(err)
	}
	handler.StartAPIHandler(handlers.DeclineFriendRequest)
}
