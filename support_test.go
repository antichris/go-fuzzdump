package fuzzdump_test

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_predicateErrWriter_Write(t *testing.T) {
	type bs = []byte
	tests := map[string]struct {
		io.Writer
		pResult bool
		err     string
		wErr    string
	}{"no error": {
		Writer:  io.Discard,
		pResult: false,
		wErr:    "",
	}, "own error": {
		Writer:  io.Discard,
		pResult: true,
		err:     snap,
		wErr:    snap,
	}, "passed error": {
		Writer:  TextErrWriter(snap),
		pResult: false,
		wErr:    snap,
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			predicate := func(bs) bool { return tt.pResult }
			w := PredicateTextErrWriter(tt.Writer, tt.err, predicate)
			_, err := w.Write(bs{})

			req := require.New(t)
			if tt.wErr != "" {
				req.EqualError(err, tt.wErr)
				return
			}
			req.NoError(err)
		})
	}
}

// TextErrWriter returns errText error on all Write calls.
func TextErrWriter(errText string) io.Writer {
	return ErrWriter(errors.New(errText))
}

// ErrWriter returns err on all Write calls.
func ErrWriter(err error) io.Writer {
	return PredicateErrWriter(nil, err, func(b []byte) bool { return true })
}

// PredicateTextErrWriter returns errText error on those Write calls for
// which the predicate returns true.
// The predicate gets passed the b to be written, so that can be used in
// its decision process.
func PredicateTextErrWriter(
	w io.Writer, errText string, predicate func([]byte) bool,
) io.Writer {
	return PredicateErrWriter(w, errors.New(errText), predicate)
}

// PredicateErrWriter returns err on those Write calls for which the
// predicate returns true.
// The predicate gets passed the b to be written, so that can be used in
// its decision process.
func PredicateErrWriter(
	w io.Writer, err error, predicate func([]byte) bool,
) io.Writer {
	return &predicateErrWriter{w, err, predicate}
}

type predicateErrWriter struct {
	io.Writer
	error
	predicate func([]byte) bool
}

// Write implements the [io.Writer] interface.
func (w *predicateErrWriter) Write(b []byte) (int, error) {
	if w.predicate(b) {
		return 0, w.error
	}
	return w.Writer.Write(b)
}
