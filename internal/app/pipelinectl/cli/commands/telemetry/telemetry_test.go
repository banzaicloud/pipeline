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

package telemetry

import (
	"flag"
	"io/ioutil"
	"sort"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

// nolint: gochecknoglobals
var update = flag.Bool("update", false, "update .golden files")

func TestParseTelemetryFromFile(t *testing.T) {
	golden := "testdata/telemetry.golden"

	telemetry, err := getTelemetry(options{
		telemetryUrl: "file://testdata/telemetry.example",
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	sort.Slice(telemetry, func(i, j int) bool {
		return telemetry[i].Name < telemetry[j].Name
	})

	tb, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(telemetry)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if *update {
		t.Logf("update %s", golden)
		if err := ioutil.WriteFile(golden, tb, 0644); err != nil {
			t.Fatalf("failed to update golden: %+v", err)
		}
	}

	goldenContent, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	assert.Equal(t, string(goldenContent), string(tb))
}
