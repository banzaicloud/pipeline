/* Copyright 2016 The Bazel Authors. All rights reserved.

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

// Command fetch_repo downloads a Go module or repository at a specific
// version or commit.
//
// In module mode, fetch_repo downloads a module using "go mod download",
// verifies the contents against a sum, then copies the contents of the module
// to a target directory. fetch_repo respects GOPATH, GOCACHE, and GOPROXY.
//
// In repository mode, fetch_repo clones a repository using a VCS tool.
// fetch_repo performs import path redirection in this mode.
package main

import (
	"flag"
	"log"

	"golang.org/x/tools/go/vcs"
)

var (
	// Common flags
	importpath = flag.String("importpath", "", "Go importpath to the repository fetch")
	dest       = flag.String("dest", "", "destination directory")

	// Repository flags
	remote = flag.String("remote", "", "The URI of the remote repository. Must be used with the --vcs flag.")
	cmd    = flag.String("vcs", "", "Version control system to use to fetch the repository. Should be one of: git,hg,svn,bzr. Must be used with the --remote flag.")
	rev    = flag.String("rev", "", "target revision")

	// Module flags
	version = flag.String("version", "", "module version. Must be semantic version or pseudo-version.")
	sum     = flag.String("sum", "", "hash of module contents")
)

// Override in tests to disable network calls.
var repoRootForImportPath = vcs.RepoRootForImportPath

func main() {
	log.SetFlags(0)
	log.SetPrefix("fetch_repo: ")

	flag.Parse()
	if *importpath == "" {
		log.Fatal("-importpath must be set")
	}
	if *dest == "" {
		log.Fatal("-dest must be set")
	}
	if flag.NArg() != 0 {
		log.Fatal("fetch_repo does not accept positional arguments")
	}

	if *version != "" {
		if *remote != "" {
			log.Fatal("-remote must not be set in module mode")
		}
		if *cmd != "" {
			log.Fatal("-vcs must not be set in module mode")
		}
		if *rev != "" {
			log.Fatal("-rev must not be set in module mode")
		}
		if *version == "" {
			log.Fatal("-version must be set in module mode")
		}
		if *sum == "" {
			log.Fatal("-sum must be set in module mode")
		}
		if err := fetchModule(*dest, *importpath, *version, *sum); err != nil {
			log.Fatal(err)
		}
	} else {
		if *version != "" {
			log.Fatal("-version must not be set in repository mode")
		}
		if *sum != "" {
			log.Fatal("-sum must not be set in repository mode")
		}
		if *rev == "" {
			log.Fatal("-rev must be set in repository mode")
		}
		if err := fetchRepo(*dest, *remote, *cmd, *importpath, *rev); err != nil {
			log.Fatal(err)
		}
	}
}
