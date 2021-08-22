package util

import (
	"errors"
	"strings"
)

type Plugin struct {
	Name       string
	Repository string
	Version    string
}

// NewPlugin parses plugin parameter into repository name and version
// and maps the result in a Plugin struct
//
// plugin must be formatted as owner/repo or owner/repo@version
// if no version is specified, latest will be used
func NewPlugin(plugin string) (*Plugin, error) {
	splitted := strings.Split(plugin, "@")
	repository := splitted[0]

	// Use latest if version is not specified
	version := "latest"
	if len(splitted) > 1 {
		version = splitted[1]
	}

	name, err := parseName(repository)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		Name:       name,
		Repository: repository,
		Version:    version,
	}, nil
}

// parseName parses the name of the plugin given the repository
// return an error if unable to parse the name
func parseName(repository string) (string, error) {
	splitted := strings.Split(repository, "/")

	// A plugin name must be formatted as owner/pluginName
	if len(splitted) != 2 {
		return "", errors.New("Error parsing the plugin name, it should be formatted as owner/repository")
	}

	return splitted[1], nil
}
