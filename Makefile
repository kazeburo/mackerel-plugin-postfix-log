VERSION=0.0.2
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all: mackerel-plugin-postfix-log

.PHONY: mackerel-plugin-postfix-log

bundle:
	dep ensure

update:
	dep ensure -update

mackerel-plugin-postfix-log: main.go
	go build $(LDFLAGS) -o mackerel-plugin-postfix-log

linux: main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o mackerel-plugin-postfix-log

check:
	go test ./...

fmt:
	go fmt ./...

tag:
	git tag v${VERSION}
	git push origin v${VERSION}
	git push origin master
	goreleaser --rm-dist
