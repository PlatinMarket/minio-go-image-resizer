build: linux

linux:
	GOOS=linux GOARCH=amd64 go build -o bin/resizer-amd64 main.go
