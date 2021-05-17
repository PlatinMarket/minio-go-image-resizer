# Define the build number with the latest commit hash.
BUILD := $(shell git log --format="%H" -n 1)

build: linux

image: build
	docker build . --tag resizer:${BUILD} --tag resizer:latest

linux:
	GOOS=linux GOARCH=amd64 go build -o bin/resizer-amd64 main.go
