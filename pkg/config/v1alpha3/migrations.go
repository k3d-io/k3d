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

package v1alpha3

import (
	"encoding/json"
	"fmt"
	"strings"

	configtypes "github.com/k3d-io/k3d/v5/pkg/config/types"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha2"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/k3d-io/k3d/v5/pkg/util"
)

var Migrations = map[string]func(configtypes.Config) (configtypes.Config, error){
	v1alpha2.ApiVersion: MigrateV1Alpha2,
}

func MigrateV1Alpha2(input configtypes.Config) (configtypes.Config, error) {
	l.Log().Debugln("Migrating v1alpha2 to v1alpha3")

	// nodefilters changed from `@group[index]` to `@group:index`
	nodeFilterReplacer := strings.NewReplacer(
		"[", ":", // replace opening bracket
		"]", "", // drop closing bracket
	)

	/*
	 * We're migrating matching fields between versions by marshalling to JSON and back
	 */
	injson, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	/*
	 * Migrate config of `kind: Simple`
	 */
	if input.GetKind() == "Simple" {
		cfgIntermediate := SimpleConfigIntermediateV1alpha2{}

		if err := json.Unmarshal(injson, &cfgIntermediate); err != nil {
			return nil, err
		}

		intermediateJSON, err := json.Marshal(cfgIntermediate)
		if err != nil {
			return nil, err
		}
		cfg := SimpleConfig{}
		if err := json.Unmarshal(intermediateJSON, &cfg); err != nil {
			return nil, err
		}

		// simple nodefilter changes
		cfg.Options.Runtime.Labels = []LabelWithNodeFilters{}

		for _, label := range input.(v1alpha2.SimpleConfig).Labels {
			cfg.Options.Runtime.Labels = append(cfg.Options.Runtime.Labels, LabelWithNodeFilters{
				Label:       label.Label,
				NodeFilters: util.ReplaceInAllElements(nodeFilterReplacer, label.NodeFilters),
			})
		}

		/*
		 * structural changes (e.g. added nodefilter support)
		 */

		cfg.Options.K3sOptions.ExtraArgs = []K3sArgWithNodeFilters{}

		for _, arg := range input.(v1alpha2.SimpleConfig).Options.K3sOptions.ExtraServerArgs {
			cfg.Options.K3sOptions.ExtraArgs = append(cfg.Options.K3sOptions.ExtraArgs, K3sArgWithNodeFilters{
				Arg: arg,
				NodeFilters: []string{
					"server:*",
				},
			})
		}

		for _, arg := range input.(v1alpha2.SimpleConfig).Options.K3sOptions.ExtraAgentArgs {
			cfg.Options.K3sOptions.ExtraArgs = append(cfg.Options.K3sOptions.ExtraArgs, K3sArgWithNodeFilters{
				Arg: arg,
				NodeFilters: []string{
					"agent:*",
				},
			})
		}

		if input.(v1alpha2.SimpleConfig).Registries.Create {
			cfg.Registries.Create = &SimpleConfigRegistryCreateConfig{
				Name:     fmt.Sprintf("%s-%s-registry", k3d.DefaultObjectNamePrefix, cfg.Name),
				Host:     "0.0.0.0",
				HostPort: "random",
			}
		}

		/*
		 * Matching fields with only syntactical changes (e.g. nodefilter syntax changed)
		 */
		for _, env := range cfg.Env {
			env.NodeFilters = util.ReplaceInAllElements(nodeFilterReplacer, env.NodeFilters)
		}

		for _, vol := range cfg.Volumes {
			vol.NodeFilters = util.ReplaceInAllElements(nodeFilterReplacer, vol.NodeFilters)
		}

		for _, p := range cfg.Ports {
			p.NodeFilters = util.ReplaceInAllElements(nodeFilterReplacer, p.NodeFilters)
		}

		/*
		 * Finalizing
		 */

		cfg.APIVersion = ApiVersion

		l.Log().Debugf("Migrated config: %+v", cfg)

		return cfg, nil
	}

	l.Log().Debugf("No migration needed for %s#%s -> %s#%s", input.GetAPIVersion(), input.GetKind(), ApiVersion, input.GetKind())

	return input, nil
}
