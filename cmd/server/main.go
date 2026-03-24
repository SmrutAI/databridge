package main

import (
	"os"

	"github.com/SmrutAI/ingestion-pipeline/internal/flow"
	"github.com/SmrutAI/ingestion-pipeline/server"
)

func main() {
	registry := flow.NewFlowRegistry()
	e := server.NewApp(registry, nil)

	port := os.Getenv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	if err := e.Start(":" + port); err != nil {
		e.Logger.Fatal(err)
	}
}
