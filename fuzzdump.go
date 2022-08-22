// Package fuzzdump implements dumping a generated Go fuzzing corpus.
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
package fuzzdump

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
)

// DumpDir writes the entries from a fuzz test corpus directory to w.
//
// It uses the first valid corpus entry it encounters to determine the
// number of fuzz arguments all entries should provide, and, consequently,
// whether to format the output as a single or multiple argument corpus.
//
// If the directory is empty, it returns [ErrEmptyCorpus].
//
// An entry with a different number of arguments than initially detected
// is not dumped, but reported with an [ErrInconsistentArgCount] in
// [CorpusErrors] returned after all files in the directory have been
// processed.
//
// If any validation errors (such as [ErrMalformedEntry] or
// [ErrUnsupportedVersion]) occurred during the parsing of the directory
// contents, a [CorpusErrors] listing the respective errors is returned
// after all files in the directory have been processed.
//
// If no valid corpus files were found, it returns an [ErrEmptyCorpus]
// wrapped in [CorpusErrors], along with all the validation errors that
// occurred.
//
// If any other error occurred during an I/O operation, it may be
// wrapped by a [fmt.Errorf].
//
// Do use [errors.Is] when checking the returned errors.
func DumpDir(w io.Writer, fsys fs.FS, dir string) (err error) {
	var errs CorpusErrors

	files, err := corpusFiles(fsys, dir)
	if err != nil {
		return err
	}
	lines, files, err := firstValidFileLines(fsys, dir, files)
	if e := errs.Capture(err); e != nil {
		return e
	}

	seps := sigleArgSep
	argCount := len(lines)
	if argCount > 1 {
		seps = multiArgSep
	}

	if _, err := fmt.Fprintln(w, seps.Pre); err != nil {
		return writeErr(err)
	}
	if err := dumpLines(w, lines); err != nil {
		return err
	}
	// Since the above already dumped the first file, we skip that one.
	err = dumpFiles(w, fsys, dir, files[1:], argCount)
	if e := errs.Capture(err); e != nil {
		return e
	}
	if _, err := fmt.Fprintln(w, seps.Post); err != nil {
		return writeErr(err)
	}

	return errs.AsError()
}

// corpusFiles wraps [getFiles] to return [ErrEmptyCorpus] if dir has no
// files.
func corpusFiles(fsys fs.FS, dir string) (files []fs.DirEntry, err error) {
	files, err = getFiles(fsys, dir)
	if err != nil {
		return
	}
	if len(files) == 0 {
		err = ErrEmptyCorpus
	}
	return
}

// firstValidFileLines returns the lines of the first valid fuzz corpus
// file and a subslice of files starting at that file.
func firstValidFileLines(
	fsys fs.FS, dir string, allFiles []fs.DirEntry,
) (lines [][]byte, files []fs.DirEntry, err error) {
	var errs CorpusErrors
	i := 0
	l := len(allFiles)
	for ; i < l; i++ {
		name := allFiles[i].Name()
		lines, err = readLines(fsys, path.Join(dir, name))
		if err == nil {
			break // The first valid corpus file has been found.
		}
		if err = errs.Capture(readErr(err, name)); err != nil {
			return
		}
	}
	if i == l {
		err = errs.Capture(ErrEmptyCorpus)
		return
	}
	files = allFiles[i:]
	err = errs.AsError()
	return
}

type separators struct{ Pre, In, Post string }

var (
	sigleArgSep = separators{Pre: "{", Post: "}"}
	multiArgSep = separators{"{{", "}, {", "}}"}
)

// dumpLines to w.
func dumpLines(w io.Writer, lines [][]byte) error {
	for _, v := range lines {
		if _, err := fmt.Fprintf(w, "\t%s,\n", v); err != nil {
			return writeErr(err)
		}
	}
	return nil
}

// dumpFiles from the given dir in fsys to w.
// In order to reduce complexity and provide more concise output, the
// expected number of fuzz arguments per corpus entry must be determined
// beforehand and passed as the value for argCount.
func dumpFiles(
	w io.Writer,
	fsys fs.FS,
	dir string,
	files []fs.DirEntry,
	argCount int,
) error {
	var errs CorpusErrors
	multiArg := argCount > 1
	for _, f := range files {
		name := f.Name()
		lines, err := readLines(fsys, path.Join(dir, name))
		if err != nil {
			if e := errs.Capture(readErr(err, name)); e != nil {
				return e
			}
			continue // Move right on to the next file.
		}
		if l := len(lines); l != argCount {
			errs.append(readErr(fmt.Errorf("%w: want %d, got %d",
				ErrInconsistentArgCount, argCount, l), name))
			continue // Skip this file.
		}
		if multiArg {
			if _, err := fmt.Fprintln(w, multiArgSep.In); err != nil {
				return writeErr(err)
			}
		}
		if err := dumpLines(w, lines); err != nil {
			return err
		}
	}
	return errs.AsError()
}

// getFiles returns those entries from dir in fsys that are regular
// files.
func getFiles(fsys fs.FS, dir string) (files []fs.DirEntry, err error) {
	s, err := fs.ReadDir(fsys, dir)
	if err != nil {
		// s is only meaningful when acquired without errors.
		return
	}
	for _, v := range s {
		if v.Type().IsRegular() {
			files = append(files, v)
		}
	}
	return
}

// readLines from file with the given name in fsys and return as a slice
// of byte slices.
func readLines(fsys fs.FS, name string) (lines [][]byte, err error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return
	}

	s := bytes.Split(b, []byte("\n"))
	if len(s) < 2 {
		// Not enough lines, so no point checking the version.
		err = ErrMalformedEntry
		return
	}
	if v := strings.TrimSuffix(string(s[0]), "\r"); v != encVersion1 {
		err = fmt.Errorf("%w: %q", ErrUnsupportedVersion, v)
		return
	}
	for _, v := range s[1:] {
		line := bytes.TrimSpace(v)
		if len(line) == 0 {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) < 1 {
		err = ErrMalformedEntry
		return
	}
	return
}

// encVersion1 is the first line of a file with version 1 encoding.
const encVersion1 = "go test fuzz v1"
