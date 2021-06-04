# dynamic config
ARG             BUILD_DATE
ARG             VCS_REF
ARG             VERSION


# web build
FROM            node:10 as web-build
WORKDIR         /app
COPY            ./web/package*.json ./web/yarn.* ./
RUN             npm install
COPY            ./web/ ./
RUN             npm run build


# go build
FROM            golang:1.16.5-alpine as go-build
RUN             apk add --update --no-cache git gcc musl-dev make
RUN             GO111MODULE=off go get github.com/gobuffalo/packr/v2/packr2
WORKDIR         /go/src/moul.io/depviz
ENV             GO111MODULE=on \
                GOPROXY=proxy.golang.org
COPY            go.* ./
RUN             go mod download
COPY            . ./
RUN             rm -rf web
COPY            --from=web-build /app/dist web
RUN             make packr
RUN             make install


# minimalist runtime
FROM alpine:3.13.5
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
COPY            --from=go-build /go/bin/depviz /bin/
ENTRYPOINT      ["depviz"]
EXPOSE          8000 9000
