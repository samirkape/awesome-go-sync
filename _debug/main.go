package main

import (
	parser "github.com/samirkape/awesome-go-sync"
	"log"
	"time"
)

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	now := time.Now()
	parser.Sync()
	elapsed := time.Since(now)
	log.Printf("Time taken to sync: %v", elapsed)
}
