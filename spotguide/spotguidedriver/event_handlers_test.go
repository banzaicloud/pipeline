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

package spotguidedriver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/spotguide"
)

type organizationCreatedHandlerStub struct {
	ctx   context.Context
	event spotguide.OrganizationCreated
}

func (s *organizationCreatedHandlerStub) OrganizationCreated(ctx context.Context, event spotguide.OrganizationCreated) error {
	s.ctx = ctx
	s.event = event

	return nil
}

func TestOrganizationCreatedEventHandler_NewEvent(t *testing.T) {
	handler := NewOrganizationCreatedEventHandler(&organizationCreatedHandlerStub{})

	event := handler.NewEvent()

	assert.IsType(t, &spotguide.OrganizationCreated{}, event)
}

func TestOrganizationCreatedEventHandler_Handle(t *testing.T) {
	h := &organizationCreatedHandlerStub{}
	handler := NewOrganizationCreatedEventHandler(h)

	ctx := context.Background()
	event := spotguide.OrganizationCreated{
		ID:     1,
		UserID: 1,
	}

	err := handler.Handle(ctx, &event)
	require.NoError(t, err)

	assert.Equal(t, h.ctx, ctx)
	assert.Equal(t, h.event, event)
}
