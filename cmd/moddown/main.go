package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	dest = flag.String("dest", "", "destination directory")
	sum  = flag.String("sum", "", "hash of module contents")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("moddown: ")

	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatalf("expected a single argument, got %d", flag.NArg())
	}

	var buf, bufErr bytes.Buffer

	cmd := exec.Command(locateGoBinary(), "mod", "download", "-x", "-modcacherw", "-json", flag.Arg(0))
	cmd.Stdout = &buf
	cmd.Stderr = io.MultiWriter(os.Stderr, &bufErr)

	err := cmd.Run()
	if err != nil { // Check if the process exited unexpectedly
		if _, ok := err.(*exec.ExitError); !ok {
			if bufErr.Len() > 0 {
				log.Fatalf("%s %s: %s", cmd.Path, strings.Join(cmd.Args, " "), bufErr.Bytes())
			} else {
				log.Fatalf("%s %s: %v", cmd.Path, strings.Join(cmd.Args, " "), err)
			}
		}
	}

	var module Module

	// Parse the JSON output
	if err := json.Unmarshal(buf.Bytes(), &module); err != nil {
		if bufErr.Len() > 0 {
			log.Fatalf("%s %s: %s", cmd.Path, strings.Join(cmd.Args, " "), bufErr.Bytes())
		} else {
			log.Fatalf("%s %s: %v", cmd.Path, strings.Join(cmd.Args, " "), err)
		}
	}

	if module.Error != "" {
		log.Fatal(module.Error)
	}

	if err != nil {
		log.Fatal(err)
	}

	if sum != nil && module.Sum != *sum {
		log.Fatalf("downloaded module with sum %s; expected sum %s", module.Sum, *sum)
	}

	if dest != nil {
		err := copyTree(*dest, module.Dir)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func locateGoBinary() string {
	if goCmd, ok := os.LookupEnv("TOOLS_GO"); ok {
		return goCmd
	}

	goCmd := "go"

	if runtime.GOOS == "windows" {
		goCmd += ".exe"
	}

	// If GOROOT is set, we'll use that one; otherwise, we'll use PATH.
	if goroot, ok := os.LookupEnv("GOROOT"); ok {
		goCmd = filepath.Join(goroot, "bin", goCmd)
	}

	return goCmd
}

// copied from fetch_repo source
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
