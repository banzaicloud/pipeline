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
	"fmt"

	"golang.org/x/tools/go/vcs"
)

func fetchRepo(dest, remote, cmd, importpath, rev string) error {
	root, err := getRepoRoot(remote, cmd, importpath)
	if err != nil {
		return err
	}
	return root.VCS.CreateAtRev(dest, root.Repo, rev)
}

func getRepoRoot(remote, cmd, importpath string) (*vcs.RepoRoot, error) {
	if (cmd == "") != (remote == "") {
		return nil, fmt.Errorf("--remote should be used with the --vcs flag. If this is an import path, use --importpath instead.")
	}

	if cmd != "" && remote != "" {
		v := vcs.ByCmd(cmd)
		if v == nil {
			return nil, fmt.Errorf("invalid VCS type: %s", cmd)
		}
		return &vcs.RepoRoot{
			VCS:  v,
			Repo: remote,
			Root: importpath,
		}, nil
	}

	// User did not give us complete information for VCS / Remote.
	// Try to figure out the information from the import path.
	verbose := false
	r, err := repoRootForImportPath(importpath, verbose)
	if err != nil {
		return nil, err
	}
	if importpath != r.Root {
		return nil, fmt.Errorf("not a root of a repository: %s", importpath)
	}
	return r, nil
}
