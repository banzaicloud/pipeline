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

package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessPointsModel_Serialization(t *testing.T) {
	m := accessPointsModel{
		{
			Name:    "ap-name-1",
			Address: "ap-address-1",
		},
		{
			Name:    "ap-name-2",
			Address: "ap-address-2",
		},
	}

	v, err := m.Value()
	require.NoError(t, err)

	var n accessPointsModel
	require.NoError(t, n.Scan(v))
	assert.Equal(t, m, n)
}

func TestAPIServerAccessPointsModel_Serialization(t *testing.T) {
	m := apiServerAccessPointsModel{
		"ap-name-1",
		"ap-name-2",
	}

	v, err := m.Value()
	require.NoError(t, err)

	var n apiServerAccessPointsModel
	require.NoError(t, n.Scan(v))
	assert.Equal(t, m, n)
}
