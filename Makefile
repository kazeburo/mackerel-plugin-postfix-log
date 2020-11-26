VERSION=0.0.5
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"
GO111MODULE=on

all: mackerel-plugin-postfix-log

.PHONY: mackerel-plugin-postfix-log

mackerel-plugin-postfix-log: main.go
	go build $(LDFLAGS) -o mackerel-plugin-postfix-log

linux: main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o mackerel-plugin-postfix-log

deps:
	go get -d
	go mod tidy

deps-update:
	go get -u -d
	go mod tidy

check:
	go test ./...

clean:
	rm -rf mackerel-plugin-postfix-log

tag:
	git tag v${VERSION}
	git push origin v${VERSION}
	git push origin master
