FROM            golang:1.11-alpine as build
RUN             apk add --update --no-cache git gcc musl-dev
ADD             . /go/src/moul.io/depviz
WORKDIR         /go/src/moul.io/depviz
RUN             GO111MODULE=on go get -v .

FROM            alpine
RUN             apk add --update --no-cache ca-certificates
COPY            --from=build /go/bin/depviz /bin/
ENTRYPOINT      ["depviz"]