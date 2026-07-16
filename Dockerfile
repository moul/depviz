FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/depviz ./cmd/depviz

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /out/depviz /usr/local/bin/depviz
ENV DEPVIZ_DB=/data/state.db \
    DEPVIZ_ADDR=0.0.0.0:8766
EXPOSE 8766
CMD ["depviz", "server"]
