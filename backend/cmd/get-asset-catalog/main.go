package main

import (
	"context"
	"log"

	"github.com/dollhouse-app/dollhouse/backend/internal/assetcatalog"
	"github.com/dollhouse-app/dollhouse/backend/internal/config"
	"github.com/dollhouse-app/dollhouse/backend/internal/handler"
)

func main() {
	assetHandler, err := assetcatalog.NewRuntimeHandler(context.Background(), config.Load())
	if err != nil {
		log.Fatal(err)
	}
	handler.StartAPIHandler(assetHandler.GetCatalog)
}
