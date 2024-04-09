//go:build mage
// +build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	//mage:import install
	"github.com/goodhosts/hostsfile/mage/install"

	//mage:import test
	"github.com/goodhosts/hostsfile/mage/test"
)

// run everything for ci process (install deps, lint, coverage, build)
func Ci() error {
	fmt.Println("Running Continuous Integration...")
	mg.Deps(
		install.Dependencies,
		Lint,
		test.Coverage,
		test.Build)
	return nil
}

// run the linter
func Lint() error {
	mg.Deps(install.Golangcilint)
	fmt.Println("Running Linter...")
	return sh.RunV("golangci-lint", "run")
}
