package utils

import "github.com/sirupsen/logrus"

type Action interface {
	GetName() string
	ExecuteAction(input interface{}) (output interface{}, err error)
}

type RevocableAction interface {
	Action
	UndoAction() (err error)
}

type ActionCallContext struct {
	Action           Action
	RemainingActions []Action
	Input            interface{}
	TryToUndo        bool
}

func NewActionCallContext(action Action, remainingActions []Action, input interface{}, tryToUndoOnFail bool) *ActionCallContext {
	return &ActionCallContext{
		Action:           action,
		RemainingActions: remainingActions,
		Input:            input,
		TryToUndo:        tryToUndoOnFail,
	}
}

func (ctx *ActionCallContext) OnCompleted(output interface{}) (interface{}, error) {
	if len(ctx.RemainingActions) == 0 {
		return output, nil
	}

	newCtx := NewActionCallContext(ctx.RemainingActions[0], ctx.RemainingActions[1:], output, ctx.TryToUndo)
	nextOutput, nextErr := newCtx.executeContextAction()
	return nextOutput, nextErr
}

func (ctx *ActionCallContext) OnFailed(error error) {
	if ctx.TryToUndo {
		revocableAction, ok := ctx.Action.(RevocableAction)
		if ok {
			revocableAction.UndoAction()
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
	} else {
		nextOutput, err := ctx.OnCompleted(selfOutput)
		if err != nil {
			ctx.OnFailed(err)
		}
		return nextOutput, err
	}
}

//--

type ActionExecutor struct {
	log *logrus.Logger
}

func NewActionExecutor(log *logrus.Logger) *ActionExecutor {
	return &ActionExecutor{
		log: log,
	}
}

func (ae *ActionExecutor) ExecuteActions(actions []Action, input interface{}, tryToUndoOnFail bool) (output interface{}, error error) {
	if len(actions) > 0 {
		action := actions[0]
		ctx := NewActionCallContext(action, actions[1:], input, tryToUndoOnFail)
		output, err := ctx.executeContextAction()
		ae.log.Info("Actions executed, success:", err == nil)
		return output, err
	}
	return input, nil
}

//func (ae *ActionExecutor) ExecuteRevocableActions(actions []RevocableAction, input interface{}) (output interface{}, error error) {
//	if len(actions) > 0 {
//		action := actions[0]
//		ctx := NewActionCallContext(action, actions[1:], input)
//		output, err := ctx.executeContextAction()
//		ae.log.Info("Actions executed, success:", err == nil)
//		return output, err
//	}
//	return input, nil
//}
