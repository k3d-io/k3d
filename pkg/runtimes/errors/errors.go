/*
Copyright © 2020-2023 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package errors

import "errors"

// ErrRuntimeNetworkNotEmpty describes an error that occurs because a network still has containers connected to it (e.g. cannot be deleted)
var ErrRuntimeNetworkNotEmpty = errors.New("network not empty")

// ErrRuntimeContainerUnknown describes the situation, where we're inspecting a container that's not obviously managed by k3d
var ErrRuntimeContainerUnknown = errors.New("container not managed by k3d: missing default label(s)")

// Runtime Network Errors
var (
	ErrRuntimeNetworkNotExists     = errors.New("network does not exist")
	ErrRuntimeNetworkMultiSameName = errors.New("multiple networks with same name found")
)

// Container Filesystem Errors
var ErrRuntimeFileNotFound = errors.New("file not found")

// Runtime Volume Errors
var (
	ErrRuntimeVolumeNotExists = errors.New("volume does not exist")
)
