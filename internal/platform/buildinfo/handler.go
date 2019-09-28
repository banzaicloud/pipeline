// Copyright © 2019 Banzai Cloud
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

package buildinfo

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/internal/global"
)

// Handler returns an HTTP handler for version information.
func Handler(buildInfo BuildInfo) http.Handler {
	var body []byte

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body == nil {
			var err error

			data := client.VersionResponse{
				Version:      buildInfo.Version,
				CommitHash:   buildInfo.CommitHash,
				BuildDate:    buildInfo.BuildDate,
				GoVersion:    buildInfo.GoVersion,
				Os:           buildInfo.Os,
				Arch:         buildInfo.Arch,
				Compiler:     buildInfo.Compiler,
				InstanceUuid: global.PipelineUUID(),
			}

			body, err = json.Marshal(data)
			if err != nil {
				panic(errors.Wrap(err, "failed to render version information"))
			}
		}

		_, _ = w.Write(body)
	})
}
