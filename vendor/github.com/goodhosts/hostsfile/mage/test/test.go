package test

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

// Unit run all unit tests
func Unit() error {
	fmt.Println("Running Tests...")
	return sh.RunV("go", "test")
}

// Build run a test build to confirm no compilation errors
func Build() error {
	fmt.Println("Running Build...")
	return sh.RunV("go", "build", "-tags", "test")
}

// Coverage run all unit tests and output coverage
func Coverage() error {
	fmt.Println("Running Tests with Coverage...")
	return sh.RunV("go", "test", "-v", "-coverprofile=coverage.txt", ".")
}

// HTML display the html coverage report from the cover tool
func HTML() error {
	if err := Coverage(); err != nil {
		return err
	}
	return sh.RunV("go", "tool", "cover", "-html", "coverage.txt")
}

// Func display the func coverage report from the cover tool
func Func() error {
	if err := Coverage(); err != nil {
		return err
	}
	return sh.RunV("go", "tool", "cover", "-func", "coverage.txt")
}
