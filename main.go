package main

import (
	"github.com/samirkape/awesome-go-sync/parser"
)

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	parser.TestTrimString()
	parser.Sync()
}
