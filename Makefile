TARGET_TYPE ?= png
WORKDIR ?= /tmp
ROADMAP_FETCH ?= 0
PREVIEW_COMMAND ?= qlmanage -p
SHELL=/bin/bash -o pipefail
SHOW_ORPHANS ?= 0
SHOW_CLOSED ?= 0
UNFLATTEN_FILTER ?= unflatten -l 10 -c 10

roadmap: roadmap_build roadmap_view

roadmap_build: install
	SHOW_ORPHANS=0 SHOW_CLOSED=0 roadmap | tee $(WORKDIR)/roadmap.dot | cat -n
	set -o pipefail; cat $(WORKDIR)/roadmap.dot | dot -T$(TARGET_TYPE) > $(WORKDIR)/roadmap.$(TARGET_TYPE)

orphans: orphans_build orphans_view

build_all: roadmap_build orphans_build everything_build

orphans_build: install
	ONLY_ORPHANS=1 SHOW_CLOSED=$(SHOW_CLOSED) roadmap | tee $(WORKDIR)/orphans.dot | cat -n
	set -o pipefail; cat $(WORKDIR)/orphans.dot | $(UNFLATTEN_FILTER) | dot -T$(TARGET_TYPE) > $(WORKDIR)/orphans.$(TARGET_TYPE)

roadmap_view:
	$(PREVIEW_COMMAND) $(WORKDIR)/roadmap.$(TARGET_TYPE) >/dev/null

orphans_view:
	$(PREVIEW_COMMAND) $(WORKDIR)/orphans.$(TARGET_TYPE) >/dev/null

dev: install
	ROADMAP_FETCH=$(ROADMAP_FETCH) SHOW_ORPHANS=$(SHOW_ORPHANS) SHOW_CLOSED=$(SHOW_CLOSED) roadmap | tee $(WORKDIR)/roadmap-dev.dot | cat -n
	set -o pipefail; cat $(WORKDIR)/roadmap-dev.dot | $(UNFLATTEN_FILTER) | dot -T$(TARGET_TYPE) > $(WORKDIR)/roadmap-dev.$(TARGET_TYPE)
	$(PREVIEW_COMMAND) $(WORKDIR)/roadmap-dev.$(TARGET_TYPE)

everything: everything_build everything_view

everything_build: install
	SHOW_ORPHANS=1 SHOW_CLOSED=1 roadmap | tee $(WORKDIR)/everything.dot | cat -n
	set -o pipefail; cat $(WORKDIR)/everything.dot | $(UNFLATTEN_FILTER) | dot -T$(TARGET_TYPE) > $(WORKDIR)/everything.$(TARGET_TYPE)

everything_view:
	$(PREVIEW_COMMAND) $(WORKDIR)/everything.$(TARGET_TYPE) >/dev/null

fetch: install
	ONLY_FETCH=1 roadmap

install:
	go install

deps:
	go get .
