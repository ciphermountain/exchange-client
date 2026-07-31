all: lint

lint:
	cd go && golangci-lint run

generate-go:
	./scripts/bundle.sh && \
	cd go && go generate ./...

generate-ts:
	./scripts/bundle.sh && \
	cd ts && npm run generate

generate-all:
	./scripts/bundle.sh && \
	cd go && go generate ./... && \
	cd ../ts && npm run generate

check-generate:
	./scripts/check-generate.sh