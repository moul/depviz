FROM    moul/protoc-gen-gotemplate:latest as pgg

FROM golang:1.18-alpine as builder
RUN     apk --no-cache add make git go rsync libc-dev openssh docker
WORKDIR /go/src/moul.io/depviz/tool/docker-protoc
ADD     go.mod go.sum tools.go ./
RUN     go mod download
RUN     go install -v \
          github.com/gogo/protobuf/protoc-gen-gogofaster \
          github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
          github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
          github.com/simplealpine/json2yaml

FROM golang:1.18-alpine
RUN     apk --no-cache add git make protobuf gcc libc-dev npm perl-utils \
 &&     mkdir -p /.cache/go-build \
 &&     chmod -R 777 /.cache \
 &&     npm install -g eclint
COPY    --from=pgg /go/bin/* /go/bin/
COPY    --from=builder /go/bin/* /go/bin/
COPY    --from=pgg /protobuf /protobuf
ENV     GOPATH=/go \
        PATH=/go/bin:${PATH} \
        GOROOT=/usr/local/go
