package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	echoadapter "github.com/awslabs/aws-lambda-go-api-proxy/echo"

	"github.com/SmrutAI/databridge/internal/flow"
	"github.com/SmrutAI/databridge/internal/store"
	"github.com/SmrutAI/databridge/server"
)

var echoLambda *echoadapter.EchoLambda

func init() {
	var jobs *store.JobStore

	dsn := os.Getenv("CODEWATCH_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "CODEWATCH_DSN not set, job tracking disabled")
	} else {
		db, err := store.New(context.Background(), dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		} else {
			if err := store.AutoMigrate(db); err != nil {
				fmt.Fprintf(os.Stderr, "auto migrate: %v\n", err)
			} else {
				jobs = store.NewJobStore(db)
			}
		}
	}

	registry := flow.NewFlowRegistry()
	e := server.NewApp(registry, jobs)
	echoLambda = echoadapter.New(e)
}

// Handler is the AWS Lambda function handler.
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return echoLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
