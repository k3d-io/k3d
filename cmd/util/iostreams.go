package util

import (
	"io"
	"os"
)

// IOStreams provides the standard names for iostreams.
// This is useful for embedding and for unit testing.
// Inconsistent and different names make it hard to read and review code
// This is based on https://github.com/kubernetes-sigs/kind/blob/main/pkg/cmd/iostreams.go, but just the nice type without the dependency
type IOStreams struct {
	// In think, os.Stdin
	In io.Reader
	// Out think, os.Stdout
	Out io.Writer
	// ErrOut think, os.Stderr
	ErrOut io.Writer
}

// StandardIOStreams returns an IOStreams from os.Stdin, os.Stdout
func StandardIOStreams() IOStreams {
	return IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}
