package fuzzdump

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyCorpus is returned when there are no files in fuzz corpus.
var ErrEmptyCorpus = errors.New("no valid fuzz corpus files in directory")

// ErrMalformedEntry is returned when a corpus entry does not have a
// supported format.
var ErrMalformedEntry = errors.New("must include version and at least one value")

// ErrUnsupportedVersion is returned when a corpus entry does not have a
// supported version header.
var ErrUnsupportedVersion = errors.New("unsupported encoding version")

// ErrInconsistentArgCount is returned when a corpus entry provides a
// different number of arguments than what was first detected.
//
// This should not occur in practice in corpus data generated by Go.
var ErrInconsistentArgCount = errors.New("inconsistent arg count in corpus " +
	"entry")

// CorpusErrors is a collection of errors found in the fuzz corpus while
// reading it from the file system.
type CorpusErrors []error

// Implements the [error] interface.
func (e CorpusErrors) Error() string {
	if e.empty() {
		return "no fuzz corpus errors"
	}
	mss := []string{"fuzz corpus has errors:"}
	for _, e := range e {
		mss = append(mss, e.Error())
	}
	return strings.Join(mss, "\n\t")
}

// Is reports whether any error in e matches target.
// Implements the interface required by [errors.Is].
//
// When target is [CorpusErrors], it returns true if both target and e
// are empty, or if e has all the errors that target has.
func (e CorpusErrors) Is(target error) bool {
	if t, ok := target.(CorpusErrors); ok {
		if ee, te := e.empty(), t.empty(); ee || te {
			// TODO Consider relaxing to true for empty t with any e.
			// Checking if either is empty here reduces Is/Unwrap calls.
			return ee == te
		}
		for _, err := range t {
			if !errors.Is(e, err) {
				return false
			}
		}
		// All errors in t match one or more in e.
		return true
	}
	return errors.Is(e.last(), target)
}

// Unwrap returns e without its last error, or nil if e is empty.
// Implements the interface required by [errors.Unwrap].
func (e CorpusErrors) Unwrap() error {
	if e.empty() {
		return nil
	}
	return e[:e.lastIndex()].AsError()
}

// AsError returns e if errors are present, otherwise it returns nil.
func (e CorpusErrors) AsError() error {
	if e.empty() {
		return nil
	}
	return e
}

// last error in e.
// Returns nil if e is empty.
func (e CorpusErrors) last() error {
	if e.empty() {
		return nil
	}
	return e[e.lastIndex()]
}

// empty returns true if there are no errors present in e.
func (e CorpusErrors) empty() bool { return len(e) == 0 }

// Unwrap returns the last error appended to e.
// Implements the interface required by [errors.Unwrap].
func (e CorpusErrors) lastIndex() int {
	return len(e) - 1
}

// Capture non-critical errors, pass critical ones.
//
// When err is one of the entry validation errors ([ErrMalformedEntry]
// or [ErrUnsupportedVersion], [ErrInconsistentArgCount]), it is
// appended to e and nil is returned.
//
// When err is [ErrEmptyCorpus], it also gets appended to e, but since
// it occurs when corpus is not usable, the whole e is returned as an
// error instead.
//
// When err is another [CorpusErrors] instance, each of the errors it
// holds is processed as above. So a single [ErrEmptyCorpus] would
// propagate aborts up the stack of [CorpusErrors.Capture]'s.
//
// Any other error is returned as it is.
func (e *CorpusErrors) Capture(err error) error {
	if err == nil {
		// We'd get the same result if we went through with the rest.
		return nil
	}
	if errs, ok := err.(CorpusErrors); ok {
		// TODO Consider appending it whole instead.
		for _, err := range errs {
			if err := e.Capture(err); err != nil {
				return err
			}
		}
		return nil
	}
	if IsValidationError(err) {
		e.append(err)
		return nil
	}
	if errors.Is(err, ErrEmptyCorpus) {
		e.append(err)
		return e.AsError()
	}
	return err
}

// append errs to e.
func (e *CorpusErrors) append(errs ...error) { *e = append(*e, errs...) }

// IsValidationError returns true if err is one of the entry validation
// errors ([ErrMalformedEntry], [ErrUnsupportedVersion] or
// [ErrInconsistentArgCount]).
func IsValidationError(err error) bool {
	return errors.Is(err, ErrMalformedEntry) ||
		errors.Is(err, ErrUnsupportedVersion) ||
		errors.Is(err, ErrInconsistentArgCount)
}

func readErr(err error, fileName string) error {
	if err != nil {
		return fmt.Errorf("reading %q: %w", fileName, err)
	}
	return nil
}

func writeErr(err error) error {
	if err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}