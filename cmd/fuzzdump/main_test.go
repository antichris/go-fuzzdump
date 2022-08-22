package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/antichris/go-fuzzdump"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_main(t *testing.T) {
	defer func(v func(mainFn) shellIfaceFn) { shellIface = v }(shellIface)
	defer func(v func(int)) { exit = v }(exit)

	const code = 127
	var (
		wOut  = os.Stdout
		wErr  = os.Stderr
		wArgs = os.Args
		wCode = code
	)

	m := newMock(t)
	shellIface = func(fn mainFn) shellIfaceFn {
		return func(out, err io.Writer, args []string) int {
			return m.MethodCalled("shellIface", out, err, args).Int(0)
		}
	}
	exit = func(c int) { m.MethodCalled("exit", c) }

	m.On("shellIface", wOut, wErr, wArgs).Return(code)
	m.On("exit", wCode)

	main()

	m.AssertExpectations(t)
}

func Test_shellIface(t *testing.T) {
	const outStr = "hello"
	var (
		stdOut = &bytes.Buffer{}
		stdErr = &bytes.Buffer{}
		args   = []string{"foo/bar", "qux"}

		wWriter = stdOut
		wArgs   = args[1:]
	)
	type test struct {
		err   error
		wOut  string
		wErr  string
		wCode int
	}
	errorCase := func(err error, code int) test {
		return test{
			err:   err,
			wErr:  "bar: " + err.Error() + "\n",
			wCode: code,
		}
	}
	tests := map[string]test{
		"empty corpus": errorCase(
			fuzzdump.ErrEmptyCorpus,
			ExitEmptyCorpus,
		), "malformed corpus": errorCase(
			fuzzdump.ErrUnsupportedVersion,
			ExitSoft,
		), "no valid files": errorCase(
			fuzzdump.CorpusErrors{
				fuzzdump.ErrMalformedEntry,
				fuzzdump.ErrEmptyCorpus,
			},
			ExitEmptyCorpus,
		), "critical error": errorCase(
			errSnap,
			ExitHard,
		), "nominal": {
			wOut:  outStr,
			wCode: ExitSuccess,
		},
	}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			stdOut.Reset()
			stdErr.Reset()

			m := newMock(t)
			mockMain := func(w io.Writer, args []string) error {
				r := m.MethodCalled("mockMain", w, args)
				fmt.Fprint(w, outStr)
				return r.Error(0)
			}
			m.On("mockMain", wWriter, wArgs).Return(tt.err)

			gotMain := shellIface(mockMain)
			gotCode := gotMain(stdOut, stdErr, args)

			req := require.New(t)
			req.True(m.AssertExpectations(t))

			if tt.wErr != "" {
				errStr := stdErr.String()
				req.Equal(tt.wErr, errStr)
			} else {
				req.Empty(stdErr.String())
			}
			req.Equal(outStr, stdOut.String())
			req.Equal(tt.wCode, gotCode)
		})
	}
}

func Test_realMain(t *testing.T) {
	stdOut := &bytes.Buffer{}

	tests := map[string]struct {
		args []string
		wOut string
		wErr error
	}{"dir not given": {
		wErr: errNoDirArg,
	}, "empty dir arg": {
		args: []string{""},
		wErr: errNoDirArg,
	}, "err from dump": {
		args: []string{"."},
		wErr: fuzzdump.ErrEmptyCorpus,
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			err := realMain(stdOut, tt.args)
			req := require.New(t)
			if tt.wErr != nil {
				req.ErrorIs(err, tt.wErr)
				return
			}
			req.NoError(err)
			req.Equal(tt.wOut, stdOut.String())
		})
	}
}

var errSnap = errors.New(snap)

const snap = "snap"

func newMock(t *testing.T) *mock.Mock {
	m := &mock.Mock{}
	m.Test(t)
	return m
}
