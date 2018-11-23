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

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// ----

var _ RevocableAction = (*MultiplyAction)(nil)

type MultiplyAction struct {
	calculationResult *CalculationResult
	multiplier        int
}

func NewMultiplyAction(result *CalculationResult, multiplier int) *MultiplyAction {
	return &MultiplyAction{
		calculationResult: result,
		multiplier:        multiplier,
	}
}

func (a *MultiplyAction) GetName() string {
	return "MultiplyAction"
}

func (a *MultiplyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	inputNum := input.(int)
	a.calculationResult.value = inputNum * a.multiplier
	fmt.Printf("EXECUTE MULTIPLY OUTPUT: %v\n", a.calculationResult.value)
	return a.calculationResult.value, nil
}

func (a *MultiplyAction) UndoAction() (err error) {
	a.calculationResult.value = a.calculationResult.value / a.multiplier
	fmt.Printf("EXECUTE UNDO MULTIPLY OUTPUT: %v\n", a.calculationResult.value)
	return nil
}

// ----

var _ RevocableAction = (*SubtractAction)(nil)

type SubtractAction struct {
	calculationResult *CalculationResult
	amount            int
}

func NewSubtractAction(calculationResult *CalculationResult, amount int) *SubtractAction {
	return &SubtractAction{
		calculationResult: calculationResult,
		amount:            amount,
	}
}

func (a *SubtractAction) GetName() string {
	return "SubtractAction"
}

func (a *SubtractAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.calculationResult.value -= a.amount
	fmt.Printf("EXECUTE SUBTRACT OUTPUT: %v\n", a.calculationResult.value)
	return a.calculationResult.value, nil
}

func (a *SubtractAction) UndoAction() (err error) {
	a.calculationResult.value += a.amount
	fmt.Printf("EXECUTE UNDO SUBTRACT OUTPUT: %v\n", a.calculationResult.value)
	return nil
}

// ----

var _ RevocableAction = (*SubtractAction)(nil)

type DivideAction struct {
	calculationResult *CalculationResult
	divider           int
}

func NewDivideAction(calculationResult *CalculationResult, divider int) *DivideAction {
	return &DivideAction{
		calculationResult: calculationResult,
		divider:           divider,
	}
}

func (a *DivideAction) GetName() string {
	return "SubtractAction"
}

func (a *DivideAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	if a.divider == 0 {
		return nil, errors.New("Can't divide by zero")
	}
	a.calculationResult.value /= a.divider

	fmt.Printf("EXECUTE DIVIDE OUTPUT: %v\n", a.calculationResult.value)
	return a.calculationResult.value, nil
}

func (a *DivideAction) UndoAction() (err error) {
	if a.divider == 0 {
		return errors.New("Can't divide by zero")
	}
	a.calculationResult.value *= a.divider
	fmt.Printf("EXECUTE UNDO DIVIDE OUTPUT: %v\n", a.calculationResult.value)
	return nil
}

type CalculationResult struct {
	value int
}

func TestEmptyRevocalbeActionsShouldPass(t *testing.T) {

	actions := []Action{}

	output, err := NewActionExecutor(logrus.New()).ExecuteActions(actions, "foobar", true)
	require.NoError(t, err, "Shouldn't happen anything")
	require.Equal(t, "foobar", output, "Result and input should be the same")
}

func TestRevocableActionsShouldFail(t *testing.T) {

	initialValue := 5
	result := &CalculationResult{
		value: initialValue,
	}
	mul := NewMultiplyAction(result, 3)
	sub := NewSubtractAction(result, 6)
	div := NewDivideAction(result, 0)

	actions := []Action{mul, sub, div}

	_, err := NewActionExecutor(logrus.New()).ExecuteActions(actions, result.value, true)
	require.Error(t, err, "Divide action should return an error")
	require.Equal(t, 5, result.value, "Result should remain 5")
}

func TestRevocableActionsShouldPass(t *testing.T) {

	initialValue := 5
	result := &CalculationResult{
		value: initialValue,
	}
	mul := NewMultiplyAction(result, 4)
	sub := NewSubtractAction(result, 6)
	div := NewDivideAction(result, 2)

	actions := []Action{mul, sub, div}

	_, err := NewActionExecutor(logrus.New()).ExecuteActions(actions, result.value, true)
	require.NoError(t, err, "Actions should run")
	require.Equal(t, 7, result.value, "Result should be 7")
}
