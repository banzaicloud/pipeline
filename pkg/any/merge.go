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

package any

import (
	"reflect"

	"emperror.dev/emperror"
	"emperror.dev/errors"
)

// Merge returns the merge of its first and second arguments.
func Merge(fst, snd Value, options ...MergeOption) (Value, error) {
	return MergeWithContext(NewMergeContext(options...), fst, snd)
}

// MustMerge returns the merge of its first and second arguments.
// It panics if there were any errors during merging.
func MustMerge(fst, snd Value, options ...MergeOption) Value {
	val, err := Merge(fst, snd, options...)
	emperror.Panic(err)
	return val
}

// MergeWithContext returns the merge of fst and snd using the specified MergeContext.
func MergeWithContext(ctx MergeContext, fst, snd Value) (Value, error) {
	if ctx.shouldCheckEquality() && reflect.DeepEqual(fst, snd) {
		return fst, nil
	}

	ctx.depth++

	fstT, sndT := reflect.TypeOf(fst), reflect.TypeOf(snd)

	for _, tp := range []typePair{{fstT, sndT}, {fstT, nil}, {nil, sndT}, {nil, nil}} {
		if ms := ctx.strategies[tp]; ms != nil {
			return ms.Merge(ctx, fst, snd)
		}
	}

	return nil, errors.Errorf("cannot merge values of type %T and %T", fst, snd)
}

// MergeContext stores merge state and configuration
type MergeContext struct {
	equalityCheck equalityCheckOption
	depth         int
	strategies    map[typePair]MergeStrategy
}

// NewMergeContext returns a new MergeContext with the specified MergeOptions applied to it.
func NewMergeContext(options ...MergeOption) MergeContext {
	ctx := MergeContext{}
	MergeOptions(options).apply(&ctx)
	return ctx
}

func (ctx MergeContext) shouldCheckEquality() bool {
	switch ctx.equalityCheck {
	case WithInitialEqualityCheck:
		return ctx.depth == 0
	case WithSubtreeEqualityChecks:
		return true
	}
	return false
}

type equalityCheckOption int

const (
	// WithoutEqualityChecks option makes the merge skip all pre-merge equality checks.
	WithoutEqualityChecks equalityCheckOption = iota
	// WithInitialEqualityCheck option makes the merge check for equality of the input values before merging them to potentially skip costly merges.
	WithInitialEqualityCheck
	// WithSubtreeEqualityChecks option makes the merge check for equality of every value pair before merging them to potentially skip costly merges.
	WithSubtreeEqualityChecks
)

func (o equalityCheckOption) apply(ctx *MergeContext) {
	ctx.equalityCheck = o
}

type typePair = [2]reflect.Type

// MergeOption represents a merge configuration option.
type MergeOption interface {
	apply(*MergeContext)
}

// MergeOptions represent a list of merge configuration options.
type MergeOptions []MergeOption

func (opts MergeOptions) apply(ctx *MergeContext) {
	for _, opt := range opts {
		opt.apply(ctx)
	}
}

// MergeStrategy represents a merge strategy.
type MergeStrategy interface {
	// Merge merges two values using the provided MergeContext.
	Merge(ctx MergeContext, fst, snd Value) (Value, error)
}

// MergeStrategyFunc adapts a function to a MergeStrategy.
type MergeStrategyFunc func(MergeContext, Value, Value) (Value, error)

// Merge merges two values by delegating to the merge strategy function.
func (fn MergeStrategyFunc) Merge(ctx MergeContext, fst, snd Value) (Value, error) {
	return fn(ctx, fst, snd)
}

// MergeStrategyOption represents a merge strategy configuration option.
type MergeStrategyOption struct {
	fstType  reflect.Type
	sndType  reflect.Type
	strategy MergeStrategy
}

func (o MergeStrategyOption) apply(ctx *MergeContext) {
	if ctx.strategies == nil {
		ctx.strategies = make(map[typePair]MergeStrategy)
	}
	ctx.strategies[typePair{o.fstType, o.sndType}] = o.strategy
}

// WithStrategy returns a MergeStrategyOption with the provided parameters.
func WithStrategy(fstType, sndType reflect.Type, strategy MergeStrategy) MergeStrategyOption {
	return MergeStrategyOption{
		fstType:  fstType,
		sndType:  sndType,
		strategy: strategy,
	}
}

type useFirstMergeStrategy bool

const (
	// UseFirst is a merge strategy that always returns the first value without modifying it.
	UseFirst useFirstMergeStrategy = true
	// UseSecond is a merge strategy that always returns the second value without modifying it.
	UseSecond useFirstMergeStrategy = false
)

func (ms useFirstMergeStrategy) Merge(_ MergeContext, fst, snd Value) (Value, error) {
	if ms {
		return fst, nil
	}
	return snd, nil
}
