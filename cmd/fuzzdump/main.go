// FuzzDump dumps the entries of a fuzz test corpus directory to the
// standard output.
//
// It takes a fuzz test corpus directory path as a required argument,
// e.g.:
//
//	$ fuzzdump ./fuzz/FuzzMyFunc
//
// The output format of a single-argument corpus is similar to a plain
// slice with the type omitted, e.g.:
//
//	{
//		int(2),
//		int(3),
//		int(5),
//		// ... etc.
//	}
//
// The output format of a multiple-argument corpus is similar to a slice
// of structs, again, with the type omitted, e.g.:
//
//	{{
//		int(8),
//		string("foo"),
//	}, {
//		int(13),
//		string("bar"),
//	}, {
//		int(21),
//		string("qux"),
//		// ... etc.
//	}}
//
// Exit status codes:
//
//	0  success,
//	1  some files were invalid, but others could be dumped,
//	2  no valid corpus files were found,
//	3  another critical error occurred.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/antichris/go-fuzzdump"
)

func main() {
	exit(shellIface(realMain)(os.Stdout, os.Stderr, os.Args))
}

var exit = os.Exit

var shellIface = func(fn mainFn) shellIfaceFn {
	return func(stdOut, stdErr io.Writer, args []string) (exitCode int) {
		if err := fn(stdOut, args[1:]); err != nil {
			fmt.Fprintln(stdErr, path.Base(args[0])+":", err)
			switch {
			case errors.Is(err, fuzzdump.ErrEmptyCorpus):
				return ExitEmptyCorpus
			case fuzzdump.IsValidationError(err):
				return ExitSoft
			default:
				return ExitHard
			}
		}
		return ExitSuccess
	}
}

func realMain(w io.Writer, args []string) error {
	if len(args) == 0 || len(args[0]) == 0 {
		return errNoDirArg
	}
	return fuzzdump.DumpDir(w, os.DirFS(args[0]), ".")
}

type (
	// A shellIfaceFn takes command line arguments and standard output
	// and error streams as [io.Writer]'s, and returns an exit code.
	shellIfaceFn func(stdOut, stdErr io.Writer, args []string) (exitCode int)
	mainFn       func(w io.Writer, args []string) error
)

const (
	ExitSuccess = iota
	ExitSoft
	ExitEmptyCorpus
	ExitHard
)

var errNoDirArg = errors.New("directory path argument required")
