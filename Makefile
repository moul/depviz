.PHONY: install
install:
	GO111MODULE=auto go install -v

.PHONY: update_examples
update_examples:
	for dir in $(wildcard examples/*); do (cd $$dir && make); done
	git commit examples -m "chore: update examples"
