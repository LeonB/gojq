package gojq

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// ValueError is an interface for errors with a value for internal function.
// Return an error implementing this interface when you want to catch error
// values (not error messages) by try-catch, just like built-in error function.
// Refer to [WithFunction] to add a custom internal function.
type ValueError interface {
	error
	Value() any
}

type expectedObjectError struct {
	v any
}

func (err *expectedObjectError) Error() string {
	return "expected an object but got: " + typeErrorPreview(err.v)
}

type expectedArrayError struct {
	v any
}

func (err *expectedArrayError) Error() string {
	return "expected an array but got: " + typeErrorPreview(err.v)
}

type iteratorError struct {
	v any
}

func (err *iteratorError) Error() string {
	return "cannot iterate over: " + typeErrorPreview(err.v)
}

type arrayIndexNegativeError struct {
	v int
}

func (err *arrayIndexNegativeError) Error() string {
	return "array index should not be negative: " + Preview(err.v)
}

type arrayIndexTooLargeError struct {
	v any
}

func (err *arrayIndexTooLargeError) Error() string {
	return "array index too large: " + Preview(err.v)
}

type objectKeyNotStringError struct {
	v any
}

func (err *objectKeyNotStringError) Error() string {
	return "expected a string for object key but got: " + typeErrorPreview(err.v)
}

type arrayIndexNotNumberError struct {
	v any
}

func (err *arrayIndexNotNumberError) Error() string {
	return "expected a number for indexing an array but got: " + typeErrorPreview(err.v)
}

type stringIndexNotNumberError struct {
	v any
}

func (err *stringIndexNotNumberError) Error() string {
	return "expected a number for indexing a string but got: " + typeErrorPreview(err.v)
}

type expectedStartEndError struct {
	v any
}

func (err *expectedStartEndError) Error() string {
	return `expected "start" and "end" for slicing but got: ` + typeErrorPreview(err.v)
}

type lengthMismatchError struct{}

func (*lengthMismatchError) Error() string {
	return "length mismatch"
}

type inputNotAllowedError struct{}

func (*inputNotAllowedError) Error() string {
	return "input(s)/0 is not allowed"
}

type funcNotFoundError struct {
	f *Func
}

func (err *funcNotFoundError) Error() string {
	return "function not defined: " + err.f.Name + "/" + strconv.Itoa(len(err.f.Args))
}

type func0TypeError struct {
	name string
	v    any
}

func (err *func0TypeError) Error() string {
	return err.name + " cannot be applied to: " + typeErrorPreview(err.v)
}

type func1TypeError struct {
	name string
	v, w any
}

func (err *func1TypeError) Error() string {
	return err.name + "(" + Preview(err.w) + ") cannot be applied to: " + typeErrorPreview(err.v)
}

type func2TypeError struct {
	name    string
	v, w, x any
}

func (err *func2TypeError) Error() string {
	return err.name + "(" + Preview(err.w) + "; " + Preview(err.x) + ") cannot be applied to: " + typeErrorPreview(err.v)
}

type func0WrapError struct {
	name string
	v    any
	err  error
}

func (err *func0WrapError) Error() string {
	return err.name + " cannot be applied to " + Preview(err.v) + ": " + err.err.Error()
}

type func1WrapError struct {
	name string
	v, w any
	err  error
}

func (err *func1WrapError) Error() string {
	return err.name + "(" + Preview(err.w) + ") cannot be applied to " + Preview(err.v) + ": " + err.err.Error()
}

type func2WrapError struct {
	name    string
	v, w, x any
	err     error
}

func (err *func2WrapError) Error() string {
	return err.name + "(" + Preview(err.w) + "; " + Preview(err.x) + ") cannot be applied to " + Preview(err.v) + ": " + err.err.Error()
}

type exitCodeError struct {
	value any
	code  int
}

func (err *exitCodeError) Error() string {
	if s, ok := err.value.(string); ok {
		return "error: " + s
	}
	return "error: " + jsonMarshal(err.value)
}

func (err *exitCodeError) Value() any {
	return err.value
}

func (err *exitCodeError) ExitCode() int {
	return err.code
}

// HaltError is an error emitted by halt and halt_error functions.
// It implements [ValueError], and if the value is nil, discard the error
// and stop the iteration. Consider a query like "1, halt, 2";
// the first value is 1, and the second value is a HaltError with nil value.
// You might think the iterator should not emit an error this case, but it
// should so that we can recognize the halt error to stop the outer loop
// of iterating input values; echo 1 2 3 | gojq "., halt".
type HaltError exitCodeError

func (err *HaltError) Error() string {
	return "halt " + (*exitCodeError)(err).Error()
}

// Value returns the value of the error. This implements [ValueError],
// but halt error is not catchable by try-catch.
func (err *HaltError) Value() any {
	return (*exitCodeError)(err).Value()
}

// ExitCode returns the exit code of the error.
func (err *HaltError) ExitCode() int {
	return (*exitCodeError)(err).ExitCode()
}

type flattenDepthError struct {
	v float64
}

func (err *flattenDepthError) Error() string {
	return "flatten depth should not be negative: " + Preview(err.v)
}

type joinTypeError struct {
	v any
}

func (err *joinTypeError) Error() string {
	return "join cannot be applied to an array including: " + typeErrorPreview(err.v)
}

type timeArrayError struct{}

func (*timeArrayError) Error() string {
	return "expected an array of 8 numbers"
}

type unaryTypeError struct {
	name string
	v    any
}

func (err *unaryTypeError) Error() string {
	return "cannot " + err.name + ": " + typeErrorPreview(err.v)
}

type binopTypeError struct {
	name string
	l, r any
}

func (err *binopTypeError) Error() string {
	return "cannot " + err.name + ": " + typeErrorPreview(err.l) + " and " + typeErrorPreview(err.r)
}

type zeroDivisionError struct {
	l, r any
}

func (err *zeroDivisionError) Error() string {
	return "cannot divide " + typeErrorPreview(err.l) + " by: " + typeErrorPreview(err.r)
}

type zeroModuloError struct {
	l, r any
}

func (err *zeroModuloError) Error() string {
	return "cannot modulo " + typeErrorPreview(err.l) + " by: " + typeErrorPreview(err.r)
}

type formatNotFoundError struct {
	n string
}

func (err *formatNotFoundError) Error() string {
	return "format not defined: " + err.n
}

type formatRowError struct {
	typ string
	v   any
}

func (err *formatRowError) Error() string {
	return "@" + err.typ + " cannot format an array including: " + typeErrorPreview(err.v)
}

type tooManyVariableValuesError struct{}

func (*tooManyVariableValuesError) Error() string {
	return "too many variable values provided"
}

type expectedVariableError struct {
	n string
}

func (err *expectedVariableError) Error() string {
	return "variable defined but not bound: " + err.n
}

type variableNotFoundError struct {
	n string
}

func (err *variableNotFoundError) Error() string {
	return "variable not defined: " + err.n
}

type variableNameError struct {
	n string
}

func (err *variableNameError) Error() string {
	return "invalid variable name: " + err.n
}

type breakError struct {
	n string
	v any
}

func (err *breakError) Error() string {
	return "label not defined: " + err.n
}

func (*breakError) ExitCode() int {
	return 3
}

type tryEndError struct {
	err error
}

func (err *tryEndError) Error() string {
	return err.err.Error()
}

type invalidPathError struct {
	v any
}

func (err *invalidPathError) Error() string {
	return "invalid path against: " + typeErrorPreview(err.v)
}

type invalidPathIterError struct {
	v any
}

func (err *invalidPathIterError) Error() string {
	return "invalid path on iterating against: " + typeErrorPreview(err.v)
}

type queryParseError struct {
	fname, contents string
	err             error
}

func (err *queryParseError) QueryParseError() (string, string, error) {
	return err.fname, err.contents, err.err
}

func (err *queryParseError) Error() string {
	return "invalid query: " + err.fname + ": " + err.err.Error()
}

type jsonParseError struct {
	fname, contents string
	err             error
}

func (err *jsonParseError) JSONParseError() (string, string, error) {
	return err.fname, err.contents, err.err
}

func (err *jsonParseError) Error() string {
	return "invalid json: " + err.fname + ": " + err.err.Error()
}

func typeErrorPreview(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case Iter:
		return "gojq.Iter"
	default:
		return TypeOf(v) + " (" + Preview(v) + ")"
	}
}

type ErrorWithLocation struct {
	Inner           error
	Location        Location
	Fname, Contents string
}

// Error is mark the struct as an error.
func (err *ErrorWithLocation) Error() string {
	inner := err.Inner.Error()
	linestr, line, column := GetLineByOffset(err.Contents, err.Location.Start)
	if linestr == "" {
		return inner
	}
	log.Println(err.Location.Start)
	log.Println(err.Location.End)

	if err.Fname != "<arg>" || ContainsNewline(err.Contents) {
		return fmt.Sprintf("%s:%d\n%s  %s",
			err.Fname, line, FormatLineInfo(linestr, line, column), inner)
	}

	return fmt.Sprintf("invalid query: %s\n    %s\n    %*c  %s",
		err.Contents, linestr, column+1, '^', inner)

	// return "expected an object but got: " + typeErrorPreview(err.v)
}

// Unwrap is used to make it work with errors.Is, errors.As.
func (e *ErrorWithLocation) Unwrap() error {
	// Return the inner error.
	return e.Inner
}

func GetLineByOffset(str string, offset int) (linestr string, line, column int) {
	ss := &stringScanner{str, 0}
	for {
		str, start, ok := ss.next()
		if !ok {
			offset -= start
			break
		}
		line++
		linestr = str
		if ss.offset >= offset {
			offset -= start
			break
		}
	}
	offset = min(max(offset-1, 0), len(linestr))
	if offset > 48 {
		skip := len(trimLastInvalidRune(linestr[:offset-48]))
		linestr = linestr[skip:]
		offset -= skip
	}
	linestr = trimLastInvalidRune(linestr[:min(64, len(linestr))])
	if offset < len(linestr) {
		offset = len(trimLastInvalidRune(linestr[:offset]))
	} else {
		offset = len(linestr)
	}
	column = runewidth.StringWidth(linestr[:offset])
	return
}

func GetLineByLine(str string, line int) (linestr string) {
	ss := &stringScanner{str, 0}
	for {
		str, _, ok := ss.next()
		if !ok {
			break
		}
		if line--; line == 0 {
			linestr = str
			break
		}
	}
	if len(linestr) > 64 {
		linestr = trimLastInvalidRune(linestr[:64])
	}
	return
}

func trimLastInvalidRune(s string) string {
	for i := len(s) - 1; i >= 0 && i > len(s)-utf8.UTFMax; i-- {
		if b := s[i]; b < utf8.RuneSelf {
			return s[:i+1]
		} else if utf8.RuneStart(b) {
			if r, _ := utf8.DecodeRuneInString(s[i:]); r == utf8.RuneError {
				return s[:i]
			}
			break
		}
	}
	return s
}

func FormatLineInfo(linestr string, line, column int) string {
	l := strconv.Itoa(line)
	return fmt.Sprintf("    %s | %s\n    %*c", l, linestr, column+len(l)+4, '^')
}

type stringScanner struct {
	str    string
	offset int
}

func (ss *stringScanner) next() (line string, start int, ok bool) {
	if ss.offset == len(ss.str) {
		return
	}
	start, ok = ss.offset, true
	line = ss.str[start:]
	i := indexNewline(line)
	if i < 0 {
		ss.offset = len(ss.str)
		return
	}
	line = line[:i]
	if strings.HasPrefix(ss.str[start+i:], "\r\n") {
		i++
	}
	ss.offset += i + 1
	return
}

// Faster than strings.ContainsAny(str, "\r\n").
func ContainsNewline(str string) bool {
	return strings.IndexByte(str, '\n') >= 0 ||
		strings.IndexByte(str, '\r') >= 0
}

// Faster than strings.IndexAny(str, "\r\n").
func indexNewline(str string) (i int) {
	if i = strings.IndexByte(str, '\n'); i >= 0 {
		str = str[:i]
	}
	if j := strings.IndexByte(str, '\r'); j >= 0 {
		i = j
	}
	return
}
