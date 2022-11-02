# dynamic config
ARG             BUILD_DATE
ARG             VCS_REF
ARG             VERSION

# web build
FROM            node:12-alpine as web-build
RUN             npm i -g npm@8
RUN             apk add --no-cache python2 g++ make
WORKDIR         /app
COPY            ./web/package*.json ./web/yarn.* ./
RUN             npm install --legacy-peer-deps
COPY            ./web/ ./

# FIXME: avoid having those ARGs, make the runtime dynamic.
ARG		        NODE_ENV=development
ARG		        API_URL
ARG		        GITHUB_CLIENT_ID
ARG		        DEFAULT_TARGETS=moul/depviz-test
RUN             GITHUB_CLIENT_ID=$GITHUB_CLIENT_ID API_URL=$API_URL NODE_ENV=$NODE_ENV DEFAULT_TARGETS=$DEFAULT_TARGETS npm run build

# go build
FROM            golang:1.19.3-alpine as go-build
RUN             apk add --update --no-cache git gcc musl-dev make
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
FROM            alpine:3.16
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
