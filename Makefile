.PHONY: install
install:
	GO111MODULE=on go install -v

.PHONY: test
test:
	go test -race -cover -v ./...

.PHONY: lint
lint:
	golangci-lint run --verbose ./...

.PHONY: update_examples
update_examples:
	for dir in $(sort $(dir $(wildcard examples/*/*))); do (cd $$dir && make); done
	@echo "now you can run:"
	@echo "    git commit examples -m \"chore: update examples\""

.PHONY: docker.build
docker.build:
	docker build -t moul/depviz .

.PHONY: release
release:
	goreleaser --snapshot --skip-publish --rm-dist
	@echo -n "Do you want to release? [y/N] " && read ans && [ $${ans:-N} = y ]
	goreleaser --rm-dist
