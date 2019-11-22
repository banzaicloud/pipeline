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

package pke

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodePoolProvider_ImplementsScanner(t *testing.T) {
	require.Implements(t, (*sql.Scanner)(nil), new(NodePoolProvider))
}

func TestNodePoolProvider_ImplementsValuer(t *testing.T) {
	require.Implements(t, (*driver.Valuer)(nil), new(NodePoolProvider))
}

func TestRoles_ImplementsScanner(t *testing.T) {
	require.Implements(t, (*sql.Scanner)(nil), new(Roles))
}

func TestRoles_ImplementsValuer(t *testing.T) {
	require.Implements(t, (*driver.Valuer)(nil), new(Roles))
}

func TestLabels_ImplementsScanner(t *testing.T) {
	require.Implements(t, (*sql.Scanner)(nil), new(Labels))
}

func TestLabels_ImplementsValuer(t *testing.T) {
	require.Implements(t, (*driver.Valuer)(nil), new(Labels))
}

func TestTaints_ImplementsScanner(t *testing.T) {
	require.Implements(t, (*sql.Scanner)(nil), new(Taints))
}

func TestTaints_ImplementsValuer(t *testing.T) {
	require.Implements(t, (*driver.Valuer)(nil), new(Taints))
}
