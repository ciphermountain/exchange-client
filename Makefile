GOPACKAGES = $(shell go list ./...)

all: lint

lint:
	golangci-lint run