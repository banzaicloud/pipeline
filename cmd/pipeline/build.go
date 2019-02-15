// Copyright © 2018 Banzai Cloud
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

package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
)

// Provisioned by ldflags
// nolint: gochecknoglobals
var (
	Version    string
	CommitHash string
	BuildDate  string
)

func VersionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":     Version,
		"go_version":  runtime.Version(),
		"commit_hash": CommitHash,
		"build_date":  BuildDate,
		"os_arch":     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})
}
