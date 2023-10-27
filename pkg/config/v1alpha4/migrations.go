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

package v1alpha4

import (
	"encoding/json"
	"fmt"

	configtypes "github.com/k3d-io/k3d/v5/pkg/config/types"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha2"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha3"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
)

var Migrations = map[string]func(configtypes.Config) (configtypes.Config, error){
	v1alpha2.ApiVersion: MigrateV1Alpha2,
	v1alpha3.ApiVersion: MigrateV1Alpha3,
}

func MigrateV1Alpha2(input configtypes.Config) (configtypes.Config, error) {
	l.Log().Debugln("Migrating v1alpha2 to v1alpha4")

	// first, migrate to v1alpha3
	input, err := v1alpha3.MigrateV1Alpha2(input)
	if err != nil {
		return nil, fmt.Errorf("error migration v1alpha2 to v1alpha3: %w", err)
	}

	// then, migrate to v1alpha4
	return MigrateV1Alpha3(input)
}

func MigrateV1Alpha3(input configtypes.Config) (configtypes.Config, error) {
	l.Log().Debugln("Migrating v1alpha3 to v1alpha4")

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
		cfgIntermediate := v1alpha3.SimpleConfig{}

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

		cfg.Name = cfgIntermediate.Name

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
