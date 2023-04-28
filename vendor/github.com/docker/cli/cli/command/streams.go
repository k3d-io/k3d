package command

import (
	"io"

	"github.com/docker/cli/cli/streams"
)

// InStream is an input stream used by the DockerCli to read user input
//
// Deprecated: Use [streams.In] instead.
type InStream = streams.In

// OutStream is an output stream used by the DockerCli to write normal program
// output.
//
// Deprecated: Use [streams.Out] instead.
type OutStream = streams.Out

// NewInStream returns a new [streams.In] from an [io.ReadCloser].
//
// Deprecated: Use [streams.NewIn] instead.
func NewInStream(in io.ReadCloser) *streams.In {
	return streams.NewIn(in)
}

// NewOutStream returns a new [streams.Out] from an [io.Writer].
//
// Deprecated: Use [streams.NewOut] instead.
func NewOutStream(out io.Writer) *streams.Out {
	return streams.NewOut(out)
}
