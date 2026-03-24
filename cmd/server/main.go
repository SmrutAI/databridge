package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SmrutAI/databridge/internal/flow"
	"github.com/SmrutAI/databridge/internal/store"
	"github.com/SmrutAI/databridge/server"
)

func main() {
	var jobs *store.JobStore

	dsn := os.Getenv("CODEWATCH_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "CODEWATCH_DSN not set, job tracking disabled")
	} else {
		db, err := store.New(context.Background(), dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		} else {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil {
				defer func() {
					if cerr := sqlDB.Close(); cerr != nil {
						fmt.Fprintf(os.Stderr, "close db: %v\n", cerr)
					}
				}()
			}
			if err := store.AutoMigrate(db); err != nil {
				fmt.Fprintf(os.Stderr, "auto migrate: %v\n", err)
			} else {
				jobs = store.NewJobStore(db)
			}
		}
	}

	registry := flow.NewFlowRegistry()
	e := server.NewApp(registry, jobs)

	port := os.Getenv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	go func() {
		if err := e.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
