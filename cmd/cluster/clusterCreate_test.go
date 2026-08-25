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
package cluster

import (
	"testing"

	"github.com/spf13/viper"

	conf "github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

func TestApplyCLIOverridesExposeNodeports(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		ppViper = viper.New()

		cfg, err := applyCLIOverrides(conf.SimpleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Ports) != 0 {
			t.Fatalf("expected no port mappings, got %+v", cfg.Ports)
		}
	})

	t.Run("maps the nodeport range to server:0 when enabled", func(t *testing.T) {
		ppViper = viper.New()
		ppViper.Set("cli.expose-nodeports", true)

		cfg, err := applyCLIOverrides(conf.SimpleConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Ports) != 1 {
			t.Fatalf("expected exactly one port mapping, got %+v", cfg.Ports)
		}

		want := k3d.DefaultNodePortRange + ":" + k3d.DefaultNodePortRange
		if cfg.Ports[0].Port != want {
			t.Errorf("expected port %q, got %q", want, cfg.Ports[0].Port)
		}
		if len(cfg.Ports[0].NodeFilters) != 1 || cfg.Ports[0].NodeFilters[0] != "server:0" {
			t.Errorf("expected node filter [server:0], got %+v", cfg.Ports[0].NodeFilters)
		}
	})
}
