//go:build tools
// +build tools

package tools

import (
	_ "github.com/cockroachdb/crlfmt"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
