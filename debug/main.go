package main

import (
	parser "github.com/samirkape/awesome-go-sync"
)

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	parser.Sync()
}
