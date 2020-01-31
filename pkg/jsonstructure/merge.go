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

package jsonstructure

import (
	"reflect"

	"github.com/banzaicloud/pipeline/pkg/any"
)

// DefaultMergeOptions returns the default merge options for JSON structure values
func DefaultMergeOptions() any.MergeOptions {
	var (
		a = reflect.TypeOf(Array(nil))
		b = reflect.TypeOf(Boolean(false))
		n = reflect.TypeOf(Number(0))
		o = reflect.TypeOf(Object(nil))
		s = reflect.TypeOf(String(""))
		u = reflect.TypeOf(nil)
	)
	return any.MergeOptions{
		any.WithStrategy(a, a, pairwiseArrayMergeStrategy{}),
		any.WithStrategy(a, b, any.UseSecond),
		any.WithStrategy(a, n, any.UseSecond),
		any.WithStrategy(a, o, any.UseSecond),
		any.WithStrategy(a, s, any.UseSecond),
		any.WithStrategy(a, u, any.UseFirst),
		any.WithStrategy(b, a, any.UseSecond),
		any.WithStrategy(b, b, any.UseSecond),
		any.WithStrategy(b, n, any.UseSecond),
		any.WithStrategy(b, o, any.UseSecond),
		any.WithStrategy(b, s, any.UseSecond),
		any.WithStrategy(b, u, any.UseFirst),
		any.WithStrategy(n, a, any.UseSecond),
		any.WithStrategy(n, b, any.UseSecond),
		any.WithStrategy(n, n, any.UseSecond),
		any.WithStrategy(n, o, any.UseSecond),
		any.WithStrategy(n, s, any.UseSecond),
		any.WithStrategy(n, u, any.UseFirst),
		any.WithStrategy(o, a, any.UseSecond),
		any.WithStrategy(o, b, any.UseSecond),
		any.WithStrategy(o, n, any.UseSecond),
		any.WithStrategy(o, o, pairwiseObjectMergeStrategy{}),
		any.WithStrategy(o, s, any.UseSecond),
		any.WithStrategy(o, u, any.UseFirst),
		any.WithStrategy(s, a, any.UseSecond),
		any.WithStrategy(s, b, any.UseSecond),
		any.WithStrategy(s, n, any.UseSecond),
		any.WithStrategy(s, o, any.UseSecond),
		any.WithStrategy(s, s, any.UseSecond),
		any.WithStrategy(s, u, any.UseFirst),
		any.WithStrategy(u, a, any.UseSecond),
		any.WithStrategy(u, b, any.UseSecond),
		any.WithStrategy(u, n, any.UseSecond),
		any.WithStrategy(u, o, any.UseSecond),
		any.WithStrategy(u, s, any.UseSecond),
		any.WithStrategy(u, u, any.UseFirst),
	}
}

func max(x, y int) int {
	if x > y {
		return x
	}

	return y
}

func min(x, y int) int {
	if x < y {
		return x
	}

	return y
}

type pairwiseArrayMergeStrategy struct {
	innerJoin bool
}

func (m pairwiseArrayMergeStrategy) Merge(ctx any.MergeContext, fst, snd any.Value) (any.Value, error) {
	fstArr, sndArr := fst.(Array), snd.(Array)
	fstLen, sndLen := len(fstArr), len(sndArr)

	resCap := max(fstLen, sndLen) // result's capacity must be enough for both arrays
	resLen := min(fstLen, sndLen) // but its initial length must be the common length of the two
	if m.innerJoin {              // but if we only consider the intersection of the two arrays
		resCap = resLen // result's capacity can be equal to its length
	}

	resArr := make(Array, resLen, resCap)

	for i := 0; i < resLen; i++ {
		var err error
		resArr[i], err = any.MergeWithContext(ctx, fstArr[i], sndArr[i])
		if err != nil {
			return nil, err
		}
	}

	if !m.innerJoin {
		if fstLen > resLen {
			resArr = append(resArr, fstArr[resLen:]...)
		} else {
			resArr = append(resArr, sndArr[resLen:]...)
		}
	}

	return resArr, nil
}

func appendArrayMerge(_ any.MergeContext, fst, snd any.Value) (any.Value, error) {
	fstArr, sndArr := fst.(Array), snd.(Array)
	resArr := make(Array, 0, len(fstArr)+len(sndArr))
	resArr = append(resArr, fstArr...)
	return append(resArr, sndArr...), nil
}

type pairwiseObjectMergeStrategy struct {
	innerJoin bool
}

func (m pairwiseObjectMergeStrategy) Merge(ctx any.MergeContext, fst, snd any.Value) (any.Value, error) {
	fstObj, sndObj := fst.(Object), snd.(Object)
	resObj := make(Object)
	if m.innerJoin {
		for k, dv := range fstObj {
			if sv, ok := sndObj[k]; ok {
				var err error
				if resObj[k], err = any.MergeWithContext(ctx, dv, sv); err != nil {
					return nil, err
				}
			}
		}
	} else {
		for k, v := range fstObj {
			resObj[k] = v
		}
		for k, v := range sndObj {
			if dv, ok := resObj[k]; ok {
				var err error
				if v, err = any.MergeWithContext(ctx, dv, v); err != nil {
					return nil, err
				}
			}
			resObj[k] = v
		}
	}
	return resObj, nil
}
