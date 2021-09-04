package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	parser "github.com/samirkape/awesome-go-sync"
)

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(parser.Sync)
}
