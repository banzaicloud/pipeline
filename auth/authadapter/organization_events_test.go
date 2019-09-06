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

package authadapter

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message/subscriber"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/auth"
)

func TestOrganizationEventDispatcher_OrganizationCreated(t *testing.T) {
	publisher := gochannel.NewGoChannel(gochannel.Config{}, watermill.NopLogger{})
	const topic = "auth"
	eventBus, _ := cqrs.NewEventBus(publisher, func(_ string) string { return topic }, &cqrs.JSONMarshaler{})

	messages, err := publisher.Subscribe(context.Background(), topic)
	require.NoError(t, err)

	eventDispatcher := NewOrganizationEventDispatcher(eventBus)

	event := auth.OrganizationCreated{
		ID:     1,
		UserID: 1,
	}

	err = eventDispatcher.OrganizationCreated(context.Background(), event)
	require.NoError(t, err)

	received, all := subscriber.BulkRead(messages, 1, time.Second)
	if !all {
		t.Fatal("no message received")
	}

	assert.Equal(t, string(received[0].Payload), "{\"ID\":1,\"UserID\":1}")
}
