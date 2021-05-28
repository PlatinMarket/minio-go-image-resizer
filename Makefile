# Define the build number with the latest commit hash.
BUILD := $(shell git log --format="%H" -n 1)

build: linux

image: build
	docker build . --tag resizer:${BUILD} --tag resizer:latest

linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/resizer-amd64 main.go

install:
	go build -o /usr/local/bin/resizer main.go
	cp thumbnail.env /usr/local/share/thumbnail.env
	cp resizer.service /etc/systemd/system/resizer.service
	-systemctl daemon-reload

uninstall:
	-(systemctl is-active resizer.service && systemctl is-enabled resizer.service && systemctl stop resizer.service);
	-systemctl daemon-reload
	rm	-f	/etc/systemd/system/resizer.service \
			/usr/local/share/thumbnail.env \
			/usr/local/bin/resizer
