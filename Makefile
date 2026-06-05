.PHONY: test run install live

test:
	go test ./...

run:
	go run ./cmd/depviz

install:
	go install ./cmd/depviz

live:
	go run ./cmd/depviz live
