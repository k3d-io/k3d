/*
Copyright © 2020 The k3d Author(s)

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

	configtypes "github.com/rancher/k3d/v4/pkg/config/types"
	"github.com/rancher/k3d/v4/pkg/config/v1alpha2"
	log "github.com/sirupsen/logrus"
)

var Migrations = map[string]func(configtypes.Config) (configtypes.Config, error){
	v1alpha2.ApiVersion: MigrateV1Alpha2,
}

func MigrateV1Alpha2(input configtypes.Config) (configtypes.Config, error) {
	log.Debugln("Migrating v1alpha2 to v1alpha3")

	injson, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	if input.GetKind() == "Simple" {
		cfg := SimpleConfig{}

		if err := json.Unmarshal(injson, &cfg); err != nil {
			return nil, err
		}

		cfg.Options.K3sOptions.ExtraArgs = []K3sArgWithNodeFilters{}

		for _, arg := range input.(v1alpha2.SimpleConfig).Options.K3sOptions.ExtraServerArgs {
			cfg.Options.K3sOptions.ExtraArgs = append(cfg.Options.K3sOptions.ExtraArgs, K3sArgWithNodeFilters{
				Arg: arg,
				NodeFilters: []string{
					"server[*]",
				},
			})
		}

		for _, arg := range input.(v1alpha2.SimpleConfig).Options.K3sOptions.ExtraAgentArgs {
			cfg.Options.K3sOptions.ExtraArgs = append(cfg.Options.K3sOptions.ExtraArgs, K3sArgWithNodeFilters{
				Arg: arg,
				NodeFilters: []string{
					"agent[*]",
				},
			})
		}

		cfg.APIVersion = ApiVersion

		log.Debugf("Migrated config: %+v", cfg)

		return cfg, nil

	}

	log.Debugf("No migration needed for %s#%s -> %s#%s", input.GetAPIVersion(), input.GetKind(), ApiVersion, input.GetKind())

	return input, nil

}
