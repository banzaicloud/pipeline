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

import "testing"

func TestSumPendingClusters(t *testing.T) {
	telemetry, err := getTelemetry(options{
		telemetryUrl: "file://testdata/pending.example",
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	sum := sumPendingClusters(telemetry)
	if sum != 1 {
		t.Fatalf("Expected one pending cluster, got %d", sum)
	}
}
