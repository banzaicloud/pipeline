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
package issue

import (
	"fmt"

	"github.com/spf13/viper"
)

type VersionInformation struct {
	Version    string
	CommitHash string
	BuildDate  string
}

func (v VersionInformation) String() string {
	return fmt.Sprintf("Version: %s\nCommit Hash: %s\nBuild Date: %s", v.Version, v.CommitHash, v.BuildDate)
}

type Issuer interface {
	CreateIssue(userID uint, organization, title, body string) error
}

func NewIssuer(version VersionInformation) (Issuer, error) {
	switch issueType := viper.GetString("issue.type"); issueType {
	case "github":
		return GitHubIssuer{Version: version}, nil
	default:
		return nil, fmt.Errorf("issuer type not supported: %s", issueType)
	}
}
