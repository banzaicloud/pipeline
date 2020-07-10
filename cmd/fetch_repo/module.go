/* Copyright 2019 The Bazel Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func fetchModule(dest, importpath, version, sum string) error {
	// Check that version is a complete semantic version or pseudo-version.
	if _, ok := parse(version); !ok {
		return fmt.Errorf("%q is not a valid semantic version", version)
	} else if isSemverPrefix(version) {
		return fmt.Errorf("-version must be a complete semantic version. %q is a prefix.", version)
	}

	// Locate the go binary. If GOROOT is set, we'll use that one; otherwise,
	// we'll use PATH.
	goPath := "go"
	if runtime.GOOS == "windows" {
		goPath += ".exe"
	}
	if goroot, ok := os.LookupEnv("GOROOT"); ok {
		goPath = filepath.Join(goroot, "bin", goPath)
	}

	// Check whether -modcacherw is supported.
	// Assume that fetch_repo was built with the same version of Go we're running.
	modcacherw := false
	for _, tag := range build.Default.ReleaseTags {
		if tag == "go1.14" {
			modcacherw = true
			break
		}
	}

	// Download the module. In Go 1.11, this command must be run in a module,
	// so we create a dummy module in the current directory (which should be
	// empty).
	w, err := os.OpenFile("go.mod", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return fmt.Errorf("error creating temporary go.mod: %v", err)
	}
	_, err = fmt.Fprintln(w, "module example.com/temporary/module/for/fetch_repo/download")
	if err != nil {
		w.Close()
		return fmt.Errorf("error writing temporary go.mod: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("error closing temporary go.mod: %v", err)
	}

	buf := &bytes.Buffer{}
	bufErr := &bytes.Buffer{}
	cmd := exec.Command(goPath, "mod", "download", "-json")
	if modcacherw {
		cmd.Args = append(cmd.Args, "-modcacherw")
	}
	cmd.Args = append(cmd.Args, importpath+"@"+version)
	cmd.Stdout = buf
	cmd.Stderr = bufErr
	dlErr := cmd.Run()
	os.Remove("go.mod")
	if dlErr != nil {
		if _, ok := dlErr.(*exec.ExitError); !ok {
			if bufErr.Len() > 0 {
				return fmt.Errorf("%s %s: %s", cmd.Path, strings.Join(cmd.Args, " "), bufErr.Bytes())
			} else {
				return fmt.Errorf("%s %s: %v", cmd.Path, strings.Join(cmd.Args, " "), dlErr)
			}
		}
	}

	// Parse the JSON output.
	var dl struct{ Dir, Sum, Error string }
	if err := json.Unmarshal(buf.Bytes(), &dl); err != nil {
		if bufErr.Len() > 0 {
			return fmt.Errorf("%s %s: %s", cmd.Path, strings.Join(cmd.Args, " "), bufErr.Bytes())
		} else {
			return fmt.Errorf("%s %s: %v", cmd.Path, strings.Join(cmd.Args, " "), err)
		}
	}
	if dl.Error != "" {
		return errors.New(dl.Error)
	}
	if dlErr != nil {
		return dlErr
	}
	if dl.Sum != sum {
		return fmt.Errorf("downloaded module with sum %s; expected sum %s", dl.Sum, sum)
	}

	// Copy the module to the destination.
	return copyTree(dest, dl.Dir)
}

func copyTree(destRoot, srcRoot string) error {
	return filepath.Walk(srcRoot, func(src string, info os.FileInfo, e error) (err error) {
		if e != nil {
			return e
		}
		rel, err := filepath.Rel(srcRoot, src)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(destRoot, rel)

		if info.IsDir() {
			return os.Mkdir(dest, 0777)
		} else {
			r, err := os.Open(src)
			if err != nil {
				return err
			}
			defer r.Close()
			w, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer func() {
				if cerr := w.Close(); err == nil && cerr != nil {
					err = cerr
				}
			}()
			_, err = io.Copy(w, r)
			return err
		}
	})
}

// semantic version parsing functions below this point were copied from
// cmd/go/internal/semver and cmd/go/internal/modload at go1.12beta2.

// parsed returns the parsed form of a semantic version string.
type parsed struct {
	major      string
	minor      string
	patch      string
	short      string
	prerelease string
	build      string
	err        string
}

func parse(v string) (p parsed, ok bool) {
	if v == "" || v[0] != 'v' {
		p.err = "missing v prefix"
		return
	}
	p.major, v, ok = parseInt(v[1:])
	if !ok {
		p.err = "bad major version"
		return
	}
	if v == "" {
		p.minor = "0"
		p.patch = "0"
		p.short = ".0.0"
		return
	}
	if v[0] != '.' {
		p.err = "bad minor prefix"
		ok = false
		return
	}
	p.minor, v, ok = parseInt(v[1:])
	if !ok {
		p.err = "bad minor version"
		return
	}
	if v == "" {
		p.patch = "0"
		p.short = ".0"
		return
	}
	if v[0] != '.' {
		p.err = "bad patch prefix"
		ok = false
		return
	}
	p.patch, v, ok = parseInt(v[1:])
	if !ok {
		p.err = "bad patch version"
		return
	}
	if len(v) > 0 && v[0] == '-' {
		p.prerelease, v, ok = parsePrerelease(v)
		if !ok {
			p.err = "bad prerelease"
			return
		}
	}
	if len(v) > 0 && v[0] == '+' {
		p.build, v, ok = parseBuild(v)
		if !ok {
			p.err = "bad build"
			return
		}
	}
	if v != "" {
		p.err = "junk on end"
		ok = false
		return
	}
	ok = true
	return
}

func parseInt(v string) (t, rest string, ok bool) {
	if v == "" {
		return
	}
	if v[0] < '0' || '9' < v[0] {
		return
	}
	i := 1
	for i < len(v) && '0' <= v[i] && v[i] <= '9' {
		i++
	}
	if v[0] == '0' && i != 1 {
		return
	}
	return v[:i], v[i:], true
}

func parsePrerelease(v string) (t, rest string, ok bool) {
	// "A pre-release version MAY be denoted by appending a hyphen and
	// a series of dot separated identifiers immediately following the patch version.
	// Identifiers MUST comprise only ASCII alphanumerics and hyphen [0-9A-Za-z-].
	// Identifiers MUST NOT be empty. Numeric identifiers MUST NOT include leading zeroes."
	if v == "" || v[0] != '-' {
		return
	}
	i := 1
	start := 1
	for i < len(v) && v[i] != '+' {
		if !isIdentChar(v[i]) && v[i] != '.' {
			return
		}
		if v[i] == '.' {
			if start == i || isBadNum(v[start:i]) {
				return
			}
			start = i + 1
		}
		i++
	}
	if start == i || isBadNum(v[start:i]) {
		return
	}
	return v[:i], v[i:], true
}

func parseBuild(v string) (t, rest string, ok bool) {
	if v == "" || v[0] != '+' {
		return
	}
	i := 1
	start := 1
	for i < len(v) {
		if !isIdentChar(v[i]) && v[i] != '.' {
			return
		}
		if v[i] == '.' {
			if start == i {
				return
			}
			start = i + 1
		}
		i++
	}
	if start == i {
		return
	}
	return v[:i], v[i:], true
}

func isIdentChar(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '-'
}

func isBadNum(v string) bool {
	i := 0
	for i < len(v) && '0' <= v[i] && v[i] <= '9' {
		i++
	}
	return i == len(v) && i > 1 && v[0] == '0'
}

// isSemverPrefix reports whether v is a semantic version prefix: v1 or  v1.2 (not v1.2.3).
// The caller is assumed to have checked that semver.IsValid(v) is true.
func isSemverPrefix(v string) bool {
	dots := 0
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '-', '+':
			return false
		case '.':
			dots++
			if dots >= 2 {
				return false
			}
		}
	}
	return true
}
