package main

import (
	parser "github.com/samirkape/awesome-go-sync/parser/v2"
)

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	parser.Sync()
}
