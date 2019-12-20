# dynamic config
ARG             BUILD_DATE
ARG             VCS_REF
ARG             VERSION

# build
FROM            golang:1.13-alpine as build
RUN             apk add --update --no-cache git gcc musl-dev make
RUN             GO111MODULE=off go get github.com/gobuffalo/packr/v2/packr2
WORKDIR         /go/src/moul.io/depviz
ENV             GO111MODULE=on
COPY            go.* ./
RUN             go mod download
COPY            . ./
RUN             make packr
RUN             make install

# minimalist runtime
FROM alpine:3.11
LABEL           org.label-schema.build-date=$BUILD_DATE \
                org.label-schema.name="depviz" \
                org.label-schema.description="" \
                org.label-schema.url="https://moul.io/depviz/" \
                org.label-schema.vcs-ref=$VCS_REF \
                org.label-schema.vcs-url="https://github.com/moul/depviz" \
                org.label-schema.vendor="Manfred Touron" \
                org.label-schema.version=$VERSION \
                org.label-schema.schema-version="1.0" \
                org.label-schema.cmd="docker run -i -t --rm moul/depviz" \
                org.label-schema.help="docker exec -it $CONTAINER depviz --help"
RUN             apk add --update --no-cache ca-certificates
COPY            --from=build /go/bin/depviz /bin/
ENTRYPOINT      ["depviz"]
