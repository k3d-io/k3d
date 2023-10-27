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

package config

import (
	"fmt"

	types "github.com/k3d-io/k3d/v5/pkg/config/types"
)

func Migrate(config types.Config, targetVersion string) (types.Config, error) {
	migration, ok := getMigrations(targetVersion)[config.GetAPIVersion()]
	if !ok {
		return nil, fmt.Errorf("no migration possible from '%s' to '%s'", config.GetAPIVersion(), targetVersion)
	}

	cfg, err := migration(config)
	if err != nil {
		return nil, fmt.Errorf("error migrating config: %w", err)
	}

	schema, err := GetSchemaByVersion(cfg.GetAPIVersion())
	if err != nil {
		return nil, fmt.Errorf("error getting schema for config apiVersion %s: %w", cfg.GetAPIVersion(), err)
	}

	if err := ValidateSchema(cfg, schema); err != nil {
		return config, fmt.Errorf("post-migrate schema validation failed: %w", err)
	}

	return cfg, err
}
