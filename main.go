/*
Package purge-deps provides the command line app for removing all occurrences of common vendor and package cache directories.

Usage: purge-deps [/path/to/your/projects]

If no path is provided, the current directory will be used as root directory.

Flags:
  -dry      <bool>    output found directories only - do not remove

Exit codes:
 0=success
 1=execution error
 2=cli usage error

Try:
  purge-deps .
  purge-deps ~/code/web/
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Walk walks all directories in the given path with a Breadth-First-Search approach
// and cleans each matching directory.
//
// Assumptions:
// If a file match is found, the associated func runs and the whole directory is finished.
//   --> Stop walking the matched and already processed directory
func Walk(path string, tasks []Task) error {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read file entries of directory %q: %w", path, err)
	}
	// loop over files only and search for matches
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		for _, task := range tasks {
			if task.Matches(entry.Name()) {
				// the only output to stdout of this app is the full path of a processed match
				fmt.Fprintln(os.Stdout, filepath.Join(path, entry.Name()))
				// exec clean up task and bail out of this directory
				return task.Run(filepath.Join(path, entry.Name()))
			}
		}
	}
	// loop over directories and walk into them
	for _, entry := range entries {
		// `file.IsDir()` check excludes strange files like symbolic links, device files or named pipes
		// that's exactly what we need
		if entry.IsDir() {
			if err := Walk(filepath.Join(path, entry.Name()), tasks); err != nil {
				// don't wrap the error - at this point all error paths are already wrapped
				return err
			}
		}
	}
	return nil
}

var (
	successExitCode    = 0
	errorExitCode      = 1
	errorParseExitCode = 2
)

func main() {
	flagDry := flag.Bool("dry", false, "output found directories only - do not remove")
	// populate Args
	flag.Parse()

	// default to current directory
	path := "."
	args := flag.Args()
	if len(args) > 0 && args[0] != "" {
		path = args[0]
	}

	// convert given path into an absolute and clean path
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse given path %s: %v\n", path, err)
		os.Exit(errorParseExitCode)
	}

	var runners = []runner{
		{
			available: func() bool {
				_, err := exec.LookPath("composer")
				return err == nil
			},
			matches: func(s string) bool {
				return s == "composer.json"
			},
			run: func(path string) error {
				dir := filepath.Join(filepath.Dir(path), "vendor")
				if err := os.RemoveAll(dir); err != nil && os.IsNotExist(err) {
					return fmt.Errorf("failed to remove path %s: %w", dir, err)
				}
				return nil
			},
		},
		{
			available: func() bool {
				_, err := exec.LookPath("npm")
				return err == nil
			},
			matches: func(s string) bool {
				return s == "package.json"
			},
			run: func(path string) error {
				dir := filepath.Join(filepath.Dir(path), "node_modules")
				if err := os.RemoveAll(dir); err != nil && os.IsNotExist(err) {
					return fmt.Errorf("failed to remove path %s: %w", dir, err)
				}
				return nil
			},
		},
		{
			available: func() bool {
				_, err := exec.LookPath(appName("cargo"))
				return err == nil
			},
			matches: func(s string) bool {
				return s == "Cargo.toml" || s == "cargo.toml"
			},
			run: func(path string) error {
				cmd := exec.Command(appName("cargo"), "clean") // app will be found in PATH by `exec`
				cmd.Dir = filepath.Dir(path)                   // set working dir
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to run command %q: %w", cmd.String(), err)
				}
				return nil
			},
		},
		{
			available: func() bool {
				_, err := exec.LookPath(appName("dotnet"))
				return err == nil
			},
			matches: func(s string) bool {
				return strings.HasSuffix(strings.ToLower(s), ".csproj") || strings.HasSuffix(strings.ToLower(s), ".sln")
			},
			run: func(path string) error {
				cmd := exec.Command(appName("dotnet"), "clean", "--nologo") // app will be found in PATH by `exec`
				cmd.Dir = filepath.Dir(path)                                // set working dir
				if out, err := cmd.CombinedOutput(); err != nil {
					// this one fails often, because only dotnet core projects are supported
					fmt.Fprintf(os.Stderr, "failed to run command %q: %v\n%s\n", cmd.String(), err, string(out))
					return nil
				}
				return nil
			},
		},
	}
	if *flagDry {
		// replace all ops with a default print path func when flag --dry is set
		for key := range runners {
			runners[key].run = func(path string) error {
				fmt.Fprintln(os.Stdout, path)
				return nil
			}
		}
	}

	var tasks = []Task{}
	for _, r := range runners {
		// only keep runners which we have the proper dev tools installed for
		if r.Available() {
			tasks = append(tasks, r)
		}
	}

	// no tasks no worries
	if len(tasks) == 0 {
		fmt.Fprintf(os.Stderr, "no valid package managers found (tried cargo, composer, npm)\n")
		os.Exit(errorParseExitCode)
	}

	if err := Walk(absPath, tasks); err != nil {
		fmt.Fprintf(os.Stderr, "purging failed with an error: %v\n", err)
		os.Exit(errorExitCode)
	}
	if !*flagDry {
		if err := clearCachesGo(); err != nil {
			fmt.Fprintf(os.Stderr, "purging go cache failed with an error: %v\n", err)
			os.Exit(errorExitCode)
		}
		if err := clearCachesComposer(); err != nil {
			fmt.Fprintf(os.Stderr, "purging composer cache failed with an error: %v\n", err)
			os.Exit(errorExitCode)
		}
		if err := clearCachesNpm(); err != nil {
			fmt.Fprintf(os.Stderr, "purging npm cache failed with an error: %v\n", err)
			os.Exit(errorExitCode)
		}
	}
}

// Task represents a runner which executes a function when a valid match is found.
type Task interface {
	Available() bool
	Matches(string) bool
	Run(string) error
}

func clearCachesGo() error {
	cmd := exec.Command(appName("go"), "clean", "-cache")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command %q: %w", cmd.String(), err)
	}
	cmd = exec.Command(appName("go"), "clean", "-modcache")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command %q: %w", cmd.String(), err)
	}
	cmd = exec.Command(appName("go"), "clean", "-testcache")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command %q: %w", cmd.String(), err)
	}
	return nil
}

func clearCachesComposer() error {
	cmd := exec.Command("composer", "--no-interaction", "clear-cache") // app will be found in PATH by `exec`
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command %q: %w", cmd.String(), err)
	}
	return nil
}

func clearCachesNpm() error {
	cmd := exec.Command("npm", "cache", "clean", "--force")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command %q: %w", cmd.String(), err)
	}
	return nil
}

func appName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

type runner struct {
	available func() bool
	matches   func(string) bool
	run       func(string) error
}

func (r runner) Available() bool {
	return r.available()
}
func (r runner) Matches(name string) bool {
	return r.matches(name)
}
func (r runner) Run(path string) error {
	return r.run(path)
}
