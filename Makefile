GOPKG ?=	moul.io/depviz
GOBINS =	./cmd/depviz
#GOBINS += ./tools/opml-to-github-issues
#GOBINS += ./tools/sed-i-github-issues
DOCKER_IMAGE ?=	moul/depviz

# tmp fix for go>=1.14 and bolt: see https://github.com/etcd-io/bbolt/issues/187
GO_TEST_OPTS ?= -test.timeout=60s -gcflags=all=-d=checkptr=0

PRE_INSTALL_STEPS += generate
PRE_TEST_STEPS += generate
PRE_BUILD_STEPS += generate
PRE_LING_STEPS += generate
PRE_BUMPDEPS_STEPS += generate
include rules.mk

.PHONY: run
run: install
	time depviz --debug run --no-graph moul/depviz-test moul/depviz moul-bot/depviz-test moul/multipmuri moul/graphman
	time depviz --debug server --without-recovery --godmode --realm "DEV"

.PHONY: update_examples
update_examples:
	for dir in $(sort $(dir $(wildcard examples/*/*))); do (cd $$dir && make); done
	@echo "now you can run:"
	@echo "    git commit examples -m \"chore: update examples\""


##
## generate
##


PROTOS_SRC := $(wildcard ./api/*.proto) $(wildcard ./api/internal/*.proto)
GEN_DEPS := $(PROTOS_SRC) Makefile go.sum
.PHONY: generate
generate: gen.sum
gen.sum: $(GEN_DEPS)
	shasum $(GEN_DEPS) | sort > gen.sum.tmp
	@diff -q gen.sum gen.sum.tmp || ( \
	  set -xe; \
	  docker run \
		--user=`id -u` \
		--volume="`go env GOPATH`/pkg/mod:/go/pkg/mod" \
		--volume="$(PWD):/go/src/moul.io/depviz" \
		--workdir="/go/src/moul.io/depviz" \
		--entrypoint="sh" \
		--rm \
		moul/depviz-protoc:2 \
		-xec 'make generate_local'; \
		make tidy \
	)
	@rm -f gen.sum.tmp


#PROTOC_OPTS = -I `go list -m -mod=mod -f "{{.Dir}}" github.com/grpc-ecosystem/grpc-gateway`:`go list -m -mod=mod -f {{.Dir}} github.com/gogo/protobuf`/protobuf:`go list -m -mod=mod -f {{.Dir}} github.com/golang/protobuf`:./api:/protobuf
PROTOC_OPTS = -I `go list -m -mod=mod -f "{{.Dir}}" github.com/grpc-ecosystem/grpc-gateway`:./api:/protobuf
.PHONY: generate_local
generate_local:
	go mod download
	@set -e; for proto in $(PROTOS_SRC); do ( set -xe; \
	  protoc $(PROTOC_OPTS) \
		--grpc-gateway_out=logtostderr=true:"$(GOPATH)/src" \
		--gogofaster_out="plugins=grpc:$(GOPATH)/src" \
		"$$proto" \
	); done
	@# CUSTOM CHANGE
	@# due to go module versionning (currently) incompatible with automatic protobuf generated imports
	sed -i s@moul.io/depviz/@moul.io/depviz/v3/@g internal/dvserver/dvserver.pb.go
	@# END OF CUSTOM CHANGE
	go run golang.org/x/tools/cmd/goimports -w ./pkg ./cmd ./internal
	shasum $(GEN_DEPS) | sort > gen.sum.tmp
	mv gen.sum.tmp gen.sum


.PHONY: clean
clean:
	rm -f gen.sum $(wildcard */*/*.pb.go */*/*.pb.gw.go) $(wildcard out/*) $(wildcard */*/*-packr.go) $(wildcard */*/packrd/*)


.PHONY: packr
packr:
	cd internal/dvserver && go run github.com/gobuffalo/packr/v2/packr2 build && ls -la *-packr.go packrd/packed-packr.go
