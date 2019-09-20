/*
Package main provides the command line app for removing all occurences of 'node_modules' directories.

Usage: purge-npm [/path/to/your/projects]

If no path is provided, the current directory will be used as root directory.

Flags:
  -dry      <bool>    output found directories only - do not remove

Exit codes:
 0=success
 1=execution error
 2=cli usage error

Try:
  purge-npm .
  purge-npm ~/code/web/
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Walk accesses all directories in the given path with a breadth first approach
// and removes each directory named name (and its sub-directories).
func Walk(path string, name string, printOnly bool) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read content of directory %s: %w", path, err)
	}
	for _, file := range files {
		fullPath := filepath.Join(path, file.Name())

		if file.IsDir() && file.Name() == name {
			if printOnly {
				fmt.Fprintln(os.Stdout, fullPath)
			} else {
				if err := os.RemoveAll(fullPath); err != nil {
					return fmt.Errorf("failed to remove directory %s: %w", fullPath, err)
				}
			}
			// the only output to stdout of this app is the full path of a removed folder
			fmt.Fprintln(os.Stdout, fullPath)
			continue
		}
		// file.IsDir() check excludes strange files like symbolic links, device files or named pipes
		// that's exactly what we need
		if file.IsDir() {
			if err := Walk(fullPath, name, printOnly); err != nil {
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

	err = Walk(absPath, "node_modules", *flagDry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "purging failed with an error: %v\n", err)
		os.Exit(errorExitCode)
	}
}
