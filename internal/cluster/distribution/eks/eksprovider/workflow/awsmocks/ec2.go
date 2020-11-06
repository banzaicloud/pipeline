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

package awsmocks

import (
	"github.com/banzaicloud/cadence-aws-sdk/clients/ec2stub"
)

// #+testify:mock

// nolint:deadcode // Used for mock generation and only the original interface
// is referenced.
type ec2clientstub interface {
	ec2stub.Client
}
