//go:build tools
// +build tools

// Package gomodhack ensures that `go mod` can detect some required dependencies.
// This package should not be imported directly.
package gomodhack

import (

	// required by protoc.
	_ "github.com/gogo/protobuf/gogoproto"

	// required by protoc.
	_ "github.com/gogo/protobuf/types"

	// nolint:staticcheck
	_ "github.com/golang/protobuf/proto"

	// required by protoc.
	_ "github.com/golang/protobuf/ptypes/timestamp"

	// required by protoc.
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"

	// required by depaware.
	_ "github.com/tailscale/depaware/depaware"

	// required by goimports.
	_ "golang.org/x/tools/cover"

	// required by packr.
	_ "github.com/gobuffalo/packr/v2/packr2"
)
