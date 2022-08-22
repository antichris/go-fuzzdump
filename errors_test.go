// The tests here only cover the aspects of errors that are not covered
// by tests elsewhere.

package fuzzdump_test

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/antichris/go-fuzzdump"
	"github.com/stretchr/testify/require"
)

func ExampleCorpusErrors_Capture() {
	malformed := func(name string) error {
		return fmt.Errorf("parsing %q: %w", name, ErrMalformedEntry)
	}
	badVersion := func(name string) error {
		return fmt.Errorf("parsing %q: %w", name, ErrUnsupportedVersion)
	}
	fn := func() error {
		var errs CorpusErrors

		// Perform operations that return errors.
		err := malformed("foo")
		if e := errs.Capture(err); e != nil {
			return e
		}
		err = badVersion("bar")
		if e := errs.Capture(err); e != nil {
			return e
		}
		// Execution will continue, as long as the captured error is not
		// a critical one.
		fmt.Println("hello world")

		// Rinse and repeat, as needed.

		return errs.AsError()
	}
	fmt.Println(fn())

	// Output:
	// hello world
	// fuzz corpus has errors:
	// 	parsing "foo": must include version and at least one value
	// 	parsing "bar": unsupported encoding version
}

func TestCorpusErrors_Error(t *testing.T) {
	tests := map[string]struct {
		err  CorpusErrors
		want string
	}{"nil": {
		err:  nil,
		want: "no fuzz corpus errors",
	}, "snap": {
		err:  CorpusErrors{errSnap},
		want: "fuzz corpus has errors:\n\tsnap",
	}, "several": {
		err:  CorpusErrors{errSnap, errWhoops},
		want: "fuzz corpus has errors:\n\t" + snap + "\n\t" + whoops,
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			require.EqualError(t, tt.err, tt.want)
		})
	}
}

func TestCorpusErrors_Is(t *testing.T) {
	type key string
	// Plain errors (and nil).
	const (
		nilKey key = "<nil>"
		snap   key = "snap"
		whoops key = "whoops"
	)
	// CorpusErrors collections.
	const (
		errsNil        key = "CorpusErrors(<nil>)"
		errsEmpty      key = "CorpusErrors{}"
		errsSnap       key = "CorpusErrors{snap}"
		errsSnapWhoops key = "CorpusErrors{snap,whoops}"
		errsWhoops     key = "CorpusErrors{whoops}"
		errsWrapSnap   key = "CorpusErrors{&wrapError{snap}}"
	)
	// Errors wrapped by [fmt.Errorf].
	const (
		wrapSnap           key = "&wrapError{snap}"
		wrapErrsSnap       key = "&wrapError{CorpusErrors{snap}}"
		wrapErrsSnapWhoops key = "&wrapError{CorpusErrors{snap,whoops}}"
	)
	wrap := func(err error) error { return fmt.Errorf("wrapped: %w", err) }
	var (
		// CorpusErrors collections.
		es = map[key]CorpusErrors{
			errsNil:        nil,
			errsEmpty:      {},
			errsSnap:       {errSnap},
			errsSnapWhoops: {errSnap, errWhoops},
			errsWhoops:     {errWhoops},
			errsWrapSnap:   {wrap(errSnap)},
		}
		// Errors wrapped by [fmt.Errorf].
		ws = map[key]error{
			wrapSnap:           wrap(errSnap),
			wrapErrsSnap:       wrap(CorpusErrors{errSnap}),
			wrapErrsSnapWhoops: wrap(CorpusErrors{errSnap, errWhoops}),
		}
		// Target errors.
		// Encompass plain errors, CorpusErrors collections and Errors
		// wrapped by [fmt.Errorf].
		ts = map[key]error{
			nilKey: nil,
			snap:   errSnap,
			whoops: errWhoops,
		}
	)
	for k, v := range es {
		ts[k] = v
	}
	for k, v := range ws {
		ts[k] = v
	}
	// When a value for a key is not specified, it falls back to false.
	type wantMap map[key]bool
	wantEmpty := wantMap{
		errsNil:   true,
		errsEmpty: true,
	}
	tests := map[key]wantMap{
		errsNil:   wantEmpty,
		errsEmpty: wantEmpty,
		errsSnap: {
			snap:     true,
			errsSnap: true,
		},
		errsWrapSnap: {
			snap:         true,
			errsSnap:     true,
			errsWrapSnap: true,
		},
		errsSnapWhoops: {
			snap:           true,
			whoops:         true,
			errsSnap:       true,
			errsSnapWhoops: true,
			errsWhoops:     true,
		},
		wrapErrsSnap: {
			snap:         true,
			errsSnap:     true,
			wrapErrsSnap: true,
		},
		wrapErrsSnapWhoops: {
			snap:               true,
			whoops:             true,
			errsSnap:           true,
			errsSnapWhoops:     true,
			errsWhoops:         true,
			wrapErrsSnapWhoops: true,
		},
	}
	for ek, tt := range tests {
		t.Run(fmt.Sprintf("errs=%s", ek), func(t *testing.T) {
			errs := ts[ek]
			for tk, target := range ts {
				t.Run(fmt.Sprintf("target=%s", tk), func(t *testing.T) {
					want := tt[tk]
					got := errors.Is(errs, target)
					require.Equal(t, want, got)
				})
			}
		})
	}
}

func TestCorpusErrors_Unwrap(t *testing.T) {
	tests := map[string]struct {
		err  CorpusErrors
		want error
	}{
		"nil":     {},
		"snap":    {CorpusErrors{errSnap}, nil},
		"several": {CorpusErrors{errSnap, errWhoops}, CorpusErrors{errSnap}},
	}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := tt.err.Unwrap()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCorpusErrors_Capture(t *testing.T) {
	var (
		malf = ErrMalformedEntry
		ver  = ErrUnsupportedVersion
		empt = ErrEmptyCorpus
		args = ErrInconsistentArgCount
	)
	type CE = CorpusErrors
	tests := map[string]struct {
		err   error
		want  error
		wantE CE
	}{"nil": {
		err:   nil,
		want:  nil,
		wantE: nil,
	}, "CorpusErrors{errSnap}": {
		err:   CE{errSnap},
		want:  errSnap,
		wantE: nil,
	}, "CorpusErrors{ErrMalformedEntry}": {
		err:   CE{malf},
		want:  nil,
		wantE: CE{malf},
	}, "CorpusErrors{ErrUnsupportedVersion,ErrEmptyCorpus}": {
		err:   CE{ver, empt},
		want:  empt,
		wantE: CE{ver, empt},
	}, "ErrInconsistentArgCount": {
		err:   args,
		want:  nil,
		wantE: CE{args},
	}, "snap": {
		err:  errSnap,
		want: errSnap,
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			var e CE
			got := e.Capture(tt.err)
			req := require.New(t)
			req.ErrorIs(got, tt.want)
			req.Equal(tt.wantE, e)
		})
	}
	t.Run("ErrEmptyCorpus yields exactly e", func(t *testing.T) {
		e := CE{ver}
		got := e.Capture(empt)
		require.Equal(t, got, e)
	})
}

func Test_readErr(t *testing.T) {
	tests := map[string]struct {
		err  error
		name string
		want string
	}{"nil": {
		err: nil,
	}, "snap": {
		err:  errSnap,
		name: "foo",
		want: `reading "foo": snap`,
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := XreadErr(tt.err, tt.name)
			if tt.want != "" {
				require.EqualError(t, got, tt.want)
			} else {
				require.NoError(t, got)
			}
		})
	}
}

func Test_writeErr(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{"nil": {
		err: nil,
	}, "snap": {
		err:  errSnap,
		want: `writing output: snap`,
	}}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := XwriteErr(tt.err)
			if tt.want != "" {
				require.EqualError(t, got, tt.want)
			} else {
				require.NoError(t, got)
			}
		})
	}
}

var (
	errSnap   = errors.New(snap)
	errWhoops = errors.New(whoops)
)

const (
	snap   = "snap"
	whoops = "whoops"
)
