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

package utils

import "github.com/sirupsen/logrus"

// Action is a named function which can be executed
type Action interface {
	GetName() string
	ExecuteAction(input interface{}) (output interface{}, err error)
}

// RevocableAction is an Action which can be revoked
type RevocableAction interface {
	Action
	UndoAction() (err error)
}

// ActionCallContext is context in which Actions can be executed
type ActionCallContext struct {
	Action           Action
	RemainingActions []Action
	Input            interface{}
	TryToUndo        bool
}

// NewActionCallContext creates a new ActionCallContext
func NewActionCallContext(action Action, remainingActions []Action, input interface{}, tryToUndoOnFail bool) *ActionCallContext {
	return &ActionCallContext{
		Action:           action,
		RemainingActions: remainingActions,
		Input:            input,
		TryToUndo:        tryToUndoOnFail,
	}
}

// OnCompleted is called back when an action has completed
func (ctx *ActionCallContext) OnCompleted(output interface{}) (interface{}, error) {
	if len(ctx.RemainingActions) == 0 {
		return output, nil
	}

	newCtx := NewActionCallContext(ctx.RemainingActions[0], ctx.RemainingActions[1:], output, ctx.TryToUndo)
	nextOutput, nextErr := newCtx.executeContextAction()
	return nextOutput, nextErr
}

// OnFailed is called back when an action has failed
func (ctx *ActionCallContext) OnFailed(error error) {
	if ctx.TryToUndo {
		revocableAction, ok := ctx.Action.(RevocableAction)
		if ok {
			revocableAction.UndoAction() // nolint: errcheck
		}
		//else: not revocable action
	}
}

func (ctx *ActionCallContext) executeContextAction() (interface{}, error) {
	action := ctx.Action
	selfOutput, selfError := action.ExecuteAction(ctx.Input)

	if selfError != nil {
		ctx.OnFailed(selfError)
		return selfOutput, selfError
	}

	nextOutput, err := ctx.OnCompleted(selfOutput)
	if err != nil {
		ctx.OnFailed(err)
	}
	return nextOutput, err
}

//--

// ActionExecutor executes Actions
type ActionExecutor struct {
	log logrus.FieldLogger
}

// NewActionExecutor creates a new ActionExecutor
func NewActionExecutor(log logrus.FieldLogger) *ActionExecutor {
	return &ActionExecutor{
		log: log,
	}
}

// ExecuteActions executes the defined Actions
func (ae *ActionExecutor) ExecuteActions(actions []Action, input interface{}, tryToUndoOnFail bool) (output interface{}, err error) {
	if len(actions) > 0 {
		action := actions[0]
		ctx := NewActionCallContext(action, actions[1:], input, tryToUndoOnFail)
		output, err := ctx.executeContextAction()
		ae.log.Info("Actions executed, success:", err == nil)
		return output, err
	}
	return input, nil
}
