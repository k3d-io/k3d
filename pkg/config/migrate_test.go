/*
Copyright © 2020-2021 The k3d Author(s)

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
	"testing"

	"github.com/go-test/deep"
	l "github.com/rancher/k3d/v5/pkg/logger"
	"github.com/spf13/viper"
)

func TestMigrateV1Alpha2ToV1Alpha3(t *testing.T) {

	actualPath := "test_assets/config_test_simple_migration_v1alpha2.yaml"
	expectedPath := "test_assets/config_test_simple_migration_v1alpha3.yaml"

	actualViper := viper.New()
	expectedViper := viper.New()

	actualViper.SetConfigType("yaml")
	expectedViper.SetConfigType("yaml")

	actualViper.SetConfigFile(actualPath)
	expectedViper.SetConfigFile(expectedPath)

	if err := actualViper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	if err := expectedViper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	actualCfg, err := FromViper(actualViper)
	if err != nil {
		t.Fatal(err)
	}

	if actualCfg.GetAPIVersion() != DefaultConfigApiVersion {
		actualCfg, err = Migrate(actualCfg, DefaultConfigApiVersion)
		if err != nil {
			l.Log().Fatalln(err)
		}
	}

	expectedCfg, err := FromViper(expectedViper)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(actualCfg, expectedCfg); diff != nil {
		t.Fatalf("Actual\n%#v\ndoes not match expected\n%+v\nDiff:\n%#v", actualCfg, expectedCfg, diff)
	}

}
