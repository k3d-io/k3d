# go4.org/intern [![Go Reference](https://pkg.go.dev/badge/go4.org/intern.svg)](https://pkg.go.dev/go4.org/intern)

## What

Package intern lets you make smaller comparable values by boxing a larger comparable value (such as a string header) down into a single globally unique pointer.

Docs: https://pkg.go.dev/go4.org/intern

## Status

This package is mature and stable. However, it depends on the implementation details of the Go runtime. Use with care.

This package is a core, low-level package with no substantive dependencies.

We take code review, testing, dependencies, and performance seriously, similar to Go's standard library or the golang.org/x repos.

## Motivation

Package intern was initially created for [package inet.af/netaddr](https://pkg.go.dev/inet.af/netaddr).
