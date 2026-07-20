GOPACKAGES = $(shell go list ./...)

all: lint

lint:
	golangci-lint run

generate-go:
	./scripts/bundle.sh && \
	go generate ./...

check-generate:
	./scripts/check-generate.sh