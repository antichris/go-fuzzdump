# go-fuzzdump

[![godoc-badge]][godoc]
[![release-badge]][latest-release]
[![license-badge]][license]
[![goreport-badge]][goreport]

## Dump a Go fuzz corpus

A simplistic utility to dump the contents of a Go generated fuzz test corpus.

I was curious whether I could use fuzzing to improve the coverage of my existing tests by discovering more "interesting" inputs. But the format of how a fuzz corpus is cached by Go (a single separate file for every unique argument set) felt a bit unwieldy to be reviewed.

I tried looking for a ready-made solution that would fit the bill for me, but couldn't find any. So I wrote this.

See the [reference docs][godoc] for details.


## CLI

### Installation

```sh
go install github.com/antichris/go-fuzzdump/cmd/fuzzdump@latest
```

### Operation

The `fuzzdump` command takes a fuzzing corpus directory path as an argument and dumps the corpus entries it finds there to the standard output.

#### Example

```sh
$ fuzzdump ./fuzz/FuzzMyFunc
{{
	string("foo"),
	uint(8),
}, {
	string("bar"),
	uint(13),
}, {
	string("qux"),
	uint(21),
}}
```

#### Exit status

| Code | Description                                         |
|:----:|-----------------------------------------------------|
|   0  | Success                                             |
|   1  | Some files were invalid, but others could be dumped |
|   2  | No valid corpus files were found                    |
|   3  | Another critical error occurred                     |


## License

The source code of this project is released under [Mozilla Public License Version 2.0][mpl]. See [LICENSE].

[mpl]: https://www.mozilla.org/en-US/MPL/2.0/
	"Mozilla Public License, version 2.0"

[license]: LICENSE

[godoc]: https://pkg.go.dev/github.com/antichris/go-fuzzdump
[latest-release]: https://github.com/antichris/go-fuzzdump/releases/latest
[goreport]: https://goreportcard.com/report/github.com/antichris/go-fuzzdump

[godoc-badge]: https://godoc.org/github.com/antichris/go-fuzzdump?status.svg
[release-badge]: https://img.shields.io/github/release/antichris/go-fuzzdump
[license-badge]: https://img.shields.io/github/license/antichris/go-fuzzdump
[goreport-badge]: https://goreportcard.com/badge/github.com/antichris/go-fuzzdump?status.svg
