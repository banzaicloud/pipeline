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

package auditlog

import (
	"time"
)

// Entry holds all information related to an API call event.
type Entry struct {
	Time          time.Time
	CorrelationID string
	UserID        uint
	HTTP          HTTPEntry
}

// HTTPEntry contains details related to an HTTP call for an audit log entry.
type HTTPEntry struct {
	ClientIP     string
	UserAgent    string
	Method       string
	Path         string
	RequestBody  string
	StatusCode   int
	ResponseTime int
	ResponseSize int
	Errors       []string
}
