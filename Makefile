.PHONY: test run install

test:
	go test ./...

run:
	go run ./cmd/depviz

install:
	go install ./cmd/depviz
