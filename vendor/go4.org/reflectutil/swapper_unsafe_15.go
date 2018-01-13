// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !ppc64,!ppc64le,!arm64,!mips,!mipsle,!mips64,!mips64le
// +build go1.5,!js,!appengine,!safe

package reflectutil

import "unsafe"

const haveTypedMemmove = true

// typedmemmove copies a value of type t to dst from src.
//go:noescape
func typedmemmove(reflect_rtype, dst, src unsafe.Pointer)
