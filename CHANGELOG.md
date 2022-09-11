# Changelog

This project adheres to [Semantic Versioning][semver2].


## 0.2.0

### Added

- A constant string `Error` type

### Changed

- String errors from `errors.errorString` variables to constants of our own `Error` type:

	- `ErrEmptyCorpus`
	- `ErrInconsistentArgCount`
	- `ErrMalformedEntry`
	- `ErrUnsupportedVersion`

	Although this has no practical effect on the operation of the package (apart from a negligible performance boost), it now prevents any code attempting to redefine them to other values from compiling.

- The formatting of the exit status code table in README.


## 0.1.0

Initial release


[semver2]: https://semver.org/spec/v2.0.0.html
