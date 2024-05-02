GOPACKAGES = $(shell go list ./...)

all: test

lint:
	golangci-lint run

openapi:
	go get github.com/deepmap/oapi-codegen/cmd/oapi-codegen && \
	oapi-codegen --config pkg/restclient/config.yaml pkg/restclient/client.yaml > pkg/restclient/client.gen.go