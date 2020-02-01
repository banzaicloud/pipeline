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
		any.WithStrategy(a, a, PairwiseArrayMergeStrategy{}),
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
		any.WithStrategy(o, o, PairwiseObjectMergeStrategy{}),
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

// PairwiseArrayMergeStrategy is a merge strategy for JSON arrays, merging array elements with matching indices.
// If InnerJoin is true, the resulting array will contain only the result of merging elements of the arrays with matching indices;
// otherwise, the rest of the elements from the longer array are copied to the result array beyond the common, merged part.
type PairwiseArrayMergeStrategy struct {
	// InnerJoin makes the strategy merge only elements at common indices of the two arrays.
	InnerJoin bool
}

// Merge returns the combination of two JSON arrays using a pairwise merge strategy.
func (ms PairwiseArrayMergeStrategy) Merge(ctx any.MergeContext, fst, snd any.Value) (any.Value, error) {
	fstArr, sndArr := fst.(Array), snd.(Array)
	fstLen, sndLen := len(fstArr), len(sndArr)

	resCap := max(fstLen, sndLen) // result's capacity must be enough for both arrays
	resLen := min(fstLen, sndLen) // but its initial length must be the common length of the two
	if ms.InnerJoin {             // but if we only consider the intersection of the two arrays
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

	if !ms.InnerJoin {
		if fstLen > resLen {
			resArr = append(resArr, fstArr[resLen:]...)
		} else {
			resArr = append(resArr, sndArr[resLen:]...)
		}
	}

	return resArr, nil
}

// AppendArrayMergeStrategy is a merge strategy for JSON arrays, concatenating the two arrays.
type AppendArrayMergeStrategy struct {
	// SecondFirst makes the strategy copy elements from the second array to the result before elements from the first array if true.
	SecondFirst bool
}

// Merge returns the combination of two JSON arrays by concatenating them.
func (ms AppendArrayMergeStrategy) Merge(_ any.MergeContext, fst, snd any.Value) (any.Value, error) {
	fstArr, sndArr := fst.(Array), snd.(Array)
	if ms.SecondFirst {
		fstArr, sndArr = sndArr, fstArr
	}
	resArr := make(Array, 0, len(fstArr)+len(sndArr))
	resArr = append(resArr, fstArr...)
	return append(resArr, sndArr...), nil
}

// PairwiseObjectMergeStrategy is a merge strategy for JSON objects, merging object members with matching keys.
// If InnerJoin is true, the result will be the union of the intersection of the two objects, pairwise merging members with matching keys;
// otherwise, the result will be the union of the two objects, pairwise merging members with matching keys.
type PairwiseObjectMergeStrategy struct {
	InnerJoin bool
}

// Merge returns the combination of two JSON objects using a pairwise merge strategy.
func (ms PairwiseObjectMergeStrategy) Merge(ctx any.MergeContext, fst, snd any.Value) (any.Value, error) {
	fstObj, sndObj := fst.(Object), snd.(Object)

	resObj := make(Object)

	if ms.InnerJoin {
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
