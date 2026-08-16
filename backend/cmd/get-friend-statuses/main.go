package main

import (
	"context"
	"log"

	"github.com/dollhouse-app/dollhouse/backend/internal/config"
	"github.com/dollhouse-app/dollhouse/backend/internal/handler"
	"github.com/dollhouse-app/dollhouse/backend/internal/moodstatus"
)

func main() {
	handlers, err := moodstatus.NewRuntimeHandlers(context.Background(), config.Load())
	if err != nil {
		log.Fatalf("initialize friend statuses handler: %v", err)
	}
	handler.StartAPIHandler(handlers.GetFriendStatuses)
}
