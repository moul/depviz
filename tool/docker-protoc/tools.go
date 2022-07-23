//go:build tools
// +build tools

package tools

import (
	_ "github.com/gogo/protobuf/protoc-gen-gogofaster"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
	_ "github.com/simplealpine/json2yaml"
	_ "golang.org/x/tools/cmd/goimports"
)
