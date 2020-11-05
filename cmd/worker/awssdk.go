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

package main

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/banzaicloud/cadence-aws-sdk/activities/ec2"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
)

// SessionFactory returns an aws.Session based on the activity context.
type SessionFactory interface {
	Session(ctx context.Context) (*session.Session, error)
}

// RegisterAwsActivitiesWithSessionFactory registers AWS activities with a session factory that creates a session for every activity execution.
// Use this registration method if your activities will receive credentials in the context for each activity execution.
func RegisterAwsActivitiesWithSessionFactory(worker worker.Worker, sessionFactory SessionFactory) {
	worker.RegisterActivityWithOptions(ec2.NewActivitiesWithSessionFactory(sessionFactory), activity.RegisterOptions{Name: "aws-ec2-"})
}
