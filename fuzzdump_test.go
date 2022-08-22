package fuzzdump_test

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	. "github.com/antichris/go-fuzzdump"
	"github.com/stretchr/testify/require"
)

func TestDumpDir(t *testing.T) {
	const (
		multiOut = `{{
	string("foo"),
	uint(8),
}, {
	string("bar"),
	uint(13),
}}` + LF
		sigleOut = `{
	uint(3),
	uint(5),
}` + LF
	)
	tests := map[string]struct {
		dir          string
		wErr         error
		wErrContains string
		wOut         string
	}{"absent": {
		dir:  "foo",
		wErr: os.ErrNotExist,
	}, "not a corpus dir": {
		dir:  ".",
		wErr: ErrEmptyCorpus,
	}, "no files": {
		dir:  emptyDir,
		wErr: ErrEmptyCorpus,
	}, "bad dir": {
		dir:  badMultiDir,
		wErr: ErrMalformedEntry,
		wOut: multiOut,
	}, "single arg": {
		dir:  sigleDir,
		wOut: sigleOut,
	}, "multi arg": {
		dir:  multiDir,
		wOut: multiOut,
	}, "single arg in multi arg": {
		// Should not happen in practice, but is handled, if it does.
		dir:          multiInSingleDir,
		wErr:         ErrInconsistentArgCount,
		wErrContains: "want 1, got 2",
		wOut:         sigleOut,
	}, "multi arg in single arg": {
		// Should not happen in practice, but is handled, if it does.
		dir:          singleInMultiDir,
		wErr:         ErrInconsistentArgCount,
		wErrContains: "want 2, got 1",
		wOut:         multiOut,
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			w := &strings.Builder{}
			var err error
			req := require.New(t)
			req.NotPanics(func() {
				err = DumpDir(w, fsys, tt.dir)
			})
			if tt.wErr != nil {
				req.ErrorIs(err, tt.wErr)
				if tt.wErrContains != "" {
					req.ErrorContains(err, tt.wErrContains)
				}
			} else {
				req.NoError(err)
			}
			req.Equal(tt.wOut, w.String())
		})
	}
}

func TestDumpDir_OutputErrors(t *testing.T) {
	var (
		err  = errSnap
		want = XwriteErr(err).Error()
	)
	tests := []struct {
		failOn string
	}{
		{failOn: XmultiArgSep.Pre},
		{failOn: XmultiArgSep.In},
		{failOn: XmultiArgSep.Post},
		{failOn: "\tstring(\"foo\"),"},
		{failOn: "\tstring(\"bar\"),"},
	}
	for _, tt := range tests {
		n := fmt.Sprintf("fail writing=%q", tt.failOn)
		t.Run(n, func(t *testing.T) {
			p := func(b []byte) bool { return string(b) == tt.failOn+LF }
			w := PredicateErrWriter(io.Discard, err, p)
			gotErr := DumpDir(w, fsys, multiDir)
			require.EqualError(t, gotErr, want)
		})
	}
}

func Test_corpusFiles(t *testing.T) {
	t.Run("ErrEmptyCorpus", func(t *testing.T) {
		want := ErrEmptyCorpus
		dir := emptyDir
		_, err := XcorpusFiles(fsys, dir)
		require.ErrorIs(t, err, want)
	})
}

func Test_firstValidFileLines(t *testing.T) {
	t.Run("non-critical error", func(t *testing.T) {
		want := ErrMalformedEntry
		dir := badMultiDir
		_, _, err := XfirstValidFileLines(fsys, dir, fsysFiles(t, dir))
		require.ErrorIs(t, err, want)
	})
	t.Run("critical error", func(t *testing.T) {
		checkErrNotExistPassedForFiles(t, func(
			fsys fs.FS, dir string, files []fs.DirEntry,
		) error {
			_, _, err := XfirstValidFileLines(fsys, dir, files)
			return err
		})
	})
}

func Test_dumpFiles(t *testing.T) {
	t.Run("critical error", func(t *testing.T) {
		checkErrNotExistPassedForFiles(t, func(
			fsys fs.FS, dir string, files []fs.DirEntry,
		) error {
			return XdumpFiles(io.Discard, fsys, dir, files, 0)
		})
	})
}

func Test_readLines(t *testing.T) {
	tests := map[string]struct {
		name   string
		wLines string
		wErr   error
	}{"absent": {
		name: "foo",
		wErr: os.ErrNotExist,
	}, "version only": {
		name: verOnlyFile,
		wErr: ErrMalformedEntry,
	}, "bad version": {
		name: badVerFile,
		wErr: ErrUnsupportedVersion,
	}, "no args entry": {
		name: noArgsFile,
		wErr: ErrMalformedEntry,
	}, "empty args entry": {
		name: emptyArgsFile,
		wErr: ErrMalformedEntry,
	}, "nominal": {
		name:   sigleArgFile,
		wLines: "uint(3)",
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			wLines := bytes.Split([]byte(tt.wLines), []byte("\n"))
			var gotLines [][]byte
			var gotErr error
			req := require.New(t)
			req.NotPanics(func() {
				gotLines, gotErr = XreadLines(fsys, tt.name)
			})
			if tt.wErr != nil {
				req.ErrorIs(gotErr, tt.wErr)
				return
			}
			req.NoError(gotErr)
			req.Equal(wLines, gotLines)
		})
	}
}

const (
	LF = "\n"

	emptyDir = "empty"
	badFile  = "bar"

	badDir      = "bad"
	sigleDir    = "single"
	multiDir    = "multi"
	badMultiDir = "badMulti"

	multiInSingleDir = "multi-in-single"
	singleInMultiDir = "single-in-multi"

	badVerFile    = badDir + "/badVer"
	verOnlyFile   = badDir + "/verOnly"
	noArgsFile    = badDir + "/noArgs"
	emptyArgsFile = badDir + "/emptyArgs"
	sigleArgFile  = sigleDir + "/1"
)

var fsys = func() fstest.MapFS {
	const (
		sigleData1 = "\n\nuint(3)\n\n"
		sigleData2 = "uint(5)"
		multiData1 = "\n\nstring(\"foo\")\n\nuint(8)\n\n"
		multiData2 = "string(\"bar\")\nuint(13)"
	)
	return fstest.MapFS{
		emptyDir:    &fstest.MapFile{Mode: fs.ModeDir},
		badFile:     &fstest.MapFile{},
		badVerFile:  &fstest.MapFile{Data: []byte("foo" + LF)},
		verOnlyFile: &fstest.MapFile{Data: []byte(XencVersion1)},
		noArgsFile:  &fstest.MapFile{Data: []byte(XencVersion1 + LF)},

		emptyArgsFile:      corpusFile(""),
		sigleArgFile:       corpusFile(sigleData1),
		sigleDir + "/2":    corpusFile(sigleData2),
		multiDir + "/1":    corpusFile(multiData1),
		multiDir + "/2":    corpusFile(multiData2),
		badMultiDir + "/1": corpusFile(""),
		badMultiDir + "/2": corpusFile(multiData1),
		badMultiDir + "/3": corpusFile(multiData2),
		badMultiDir + "/4": corpusFile(""),

		multiInSingleDir + "/1": corpusFile(sigleData1),
		multiInSingleDir + "/2": corpusFile(multiData1),
		multiInSingleDir + "/3": corpusFile(sigleData2),
		singleInMultiDir + "/1": corpusFile(multiData1),
		singleInMultiDir + "/2": corpusFile(sigleData1),
		singleInMultiDir + "/3": corpusFile(multiData2),
	}
}()

func checkErrNotExistPassedForFiles(
	t *testing.T,
	fn func(fsys fs.FS, dir string, files []fs.DirEntry) error,
) {
	t.Helper()
	want := os.ErrNotExist
	err := fn(fstest.MapFS{}, ".", fsysFiles(t, badDir))
	require.ErrorIs(t, err, want)
}

func fsysFiles(t *testing.T, dir string) (files []fs.DirEntry) {
	t.Helper()
	files, err := XgetFiles(fsys, dir)
	if err != nil {
		t.Fatalf("getting files: %s", err)
	}
	return
}

func corpusFile(contents string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(
		XencVersion1 + LF +
			contents + LF,
	)}
}
