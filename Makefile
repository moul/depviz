GOPKG ?= 	moul.io/depviz
GOBINS ?=	. ./tools/opml-to-github-issues ./tools/sed-i-github-issues
DOCKER_IMAGE ?=	moul/depviz

all: test install

include rules.mk

.PHONY: update_examples
update_examples:
	for dir in $(sort $(dir $(wildcard examples/*/*))); do (cd $$dir && make); done
	@echo "now you can run:"
	@echo "    git commit examples -m \"chore: update examples\""
