package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	echoadapter "github.com/awslabs/aws-lambda-go-api-proxy/echo"

	"github.com/SmrutAI/ingestion-pipeline/internal/flow"
	"github.com/SmrutAI/ingestion-pipeline/server"
)

var echoLambda *echoadapter.EchoLambda

func init() {
	registry := flow.NewFlowRegistry()
	e := server.NewApp(registry, nil)
	echoLambda = echoadapter.New(e)
}

// Handler is the AWS Lambda function handler.
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return echoLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
