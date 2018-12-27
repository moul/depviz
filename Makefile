.PHONY: install
install:
	GO111MODULE=on go install -v

.PHONY: test
test:
	go test -v ./...

.PHONY: update_examples
update_examples:
	for dir in $(sort $(dir $(wildcard examples/*/*))); do (cd $$dir && make); done
	@echo "now you can run:"
	@echo "    git commit examples -m \"chore: update examples\""

.PHONY: docker.build
docker.build:
	docker build -t moul/depviz .
