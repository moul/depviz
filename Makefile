GOPKG ?= 	moul.io/depviz
GOBINS =	./cmd/depviz
#GOBINS += ./tools/opml-to-github-issues
#GOBINS += ./tools/sed-i-github-issues
DOCKER_IMAGE ?=	moul/depviz


all: test install


PRE_INSTALL_STEPS += generate
PRE_TEST_STEPS += generate
PRE_BUILD_STEPS += generate
PRE_LING_STEPS += generate
PRE_BUMPDEPS_STEPS += generate
include rules.mk

.PHONY: run
run: install
	time depviz --debug run --no-graph moul/depviz-test moul/depviz moul-bot/depviz-test moul/multipmuri moul/graphman
	time depviz --debug server --without-recovery --godmode

.PHONY: update_examples
update_examples:
	for dir in $(sort $(dir $(wildcard examples/*/*))); do (cd $$dir && make); done
	@echo "now you can run:"
	@echo "    git commit examples -m \"chore: update examples\""


##
## generate
##


PROTOS_SRC := $(wildcard ./api/*.proto) $(wildcard ./api/internal/*.proto)
GEN_SRC := $(PROTOS_SRC) Makefile
.PHONY: generate
generate: gen.sum
gen.sum: $(GEN_SRC)
	shasum $(GEN_SRC) | sort > gen.sum.tmp
	@diff -q gen.sum gen.sum.tmp || ( \
	  set -xe; \
	  GO111MODULE=on go mod vendor; \
	  docker run \
	    --user=`id -u` \
	    --volume="$(PWD):/go/src/moul.io/depviz" \
	    --workdir="/go/src/moul.io/depviz" \
	    --entrypoint="sh" \
	    --rm \
	    moul/depviz-protoc:1 \
	    -xec 'make generate_local'; \
	    make tidy \
	)


PROTOC_OPTS = -I ./vendor/github.com/grpc-ecosystem/grpc-gateway:./api:./vendor:/protobuf
.PHONY: generate_local
generate_local:
	@set -e; for proto in $(PROTOS_SRC); do ( set -xe; \
	  protoc $(PROTOC_OPTS) \
	    --grpc-gateway_out=logtostderr=true:"$(GOPATH)/src" \
	    --gogofaster_out="plugins=grpc:$(GOPATH)/src" "$$proto" \
	); done
	goimports -w ./pkg ./cmd ./internal
	shasum $(GEN_SRC) | sort > gen.sum.tmp
	mv gen.sum.tmp gen.sum


.PHONY: clean
clean:
	rm -f gen.sum $(wildcard */*/*.pb.go */*/*.pb.gw.go) $(wildcard out/*) $(wildcard */*/*-packr.go) $(wildcard */*/packrd/*)


.PHONY: packr
packr:
	GO111MODULE=off go get github.com/gobuffalo/packr/v2/packr2
	cd internal/dvserver && packr2 && ls -la *-packr.go packrd/packed-packr.go
