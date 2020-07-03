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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrivers(t *testing.T) {
	driver1 := &inmemDriver{}
	driver2 := &inmemDriver{}

	driver := Drivers{driver1, driver2}

	entry := Entry{
		Time:          time.Now(),
		CorrelationID: "cid",
		UserID:        1,
		HTTP:          HTTPEntry{},
	}

	// TODO: write test for the error path
	err := driver.Store(entry)
	require.NoError(t, err)

	assert.Equal(t, entry, driver1.entries[0])
	assert.Equal(t, entry, driver2.entries[0])
}
