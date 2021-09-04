.PHONY: build

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/main
	zip -r ~/Desktop/async-lambda.zip bin/

