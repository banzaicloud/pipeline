// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helm

import (
	"flag"
	"regexp"
	"testing"

	"github.com/banzaicloud/pipeline/internal/global"
)

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	t.Run("platform helm home", func(t *testing.T) {
		global.Config.Helm.Home = "var/cache/test"

		expected := "var/cache/test-pipeline/helm"

		env := GeneratePlatformHelmRepoEnv()
		if env.Home.String() != expected {
			t.Fatalf("expected %s got %s", expected, env.Home.String())
		}
	})
}
