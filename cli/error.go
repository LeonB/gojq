package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/itchyny/gojq"
)

type emptyError struct {
	err error
}

func (*emptyError) Error() string {
	return ""
}

func (*emptyError) isEmptyError() {}

func (err *emptyError) ExitCode() int {
	if err, ok := err.err.(interface{ ExitCode() int }); ok {
		return err.ExitCode()
	}
	return exitCodeDefaultErr
}

// Unwrap is used to make it work with errors.Is, errors.As.
func (e *emptyError) Unwrap() error {
    // Return the inner error.
    return e.err
}

type exitCodeError struct {
	code int
}

func (err *exitCodeError) Error() string {
	return "exit code: " + strconv.Itoa(err.code)
}

func (*exitCodeError) isEmptyError() {}

func (err *exitCodeError) ExitCode() int {
	return err.code
}

type flagParseError struct {
	err error
}

func (err *flagParseError) Error() string {
	return err.err.Error()
}

func (*flagParseError) ExitCode() int {
	return exitCodeFlagParseErr
}

type compileError struct {
	err error
}

func (err *compileError) Error() string {
	return "compile error: " + err.err.Error()
}

func (*compileError) ExitCode() int {
	return exitCodeCompileErr
}

type queryParseError struct {
	fname, contents string
	err             error
}

func (err *queryParseError) Error() string {
	var offset int
	var e *gojq.ParseError
	if errors.As(err.err, &e) {
		offset = e.Offset - len(e.Token) + 1
	}
	linestr, line, column := gojq.GetLineByOffset(err.contents, offset)
	if err.fname != "<arg>" || gojq.ContainsNewline(err.contents) {
		return fmt.Sprintf("invalid query: %s:%d\n%s  %s",
			err.fname, line, gojq.FormatLineInfo(linestr, line, column), err.err)
	}
	return fmt.Sprintf("invalid query: %s\n    %s\n    %*c  %s",
		err.contents, linestr, column+1, '^', err.err)
}

func (*queryParseError) ExitCode() int {
	return exitCodeCompileErr
}

type jsonParseError struct {
	fname, contents string
	line            int
	err             error
}

func (err *jsonParseError) Error() string {
	var offset int
	if err.err == io.ErrUnexpectedEOF {
		offset = len(err.contents) + 1
	} else if e, ok := err.err.(*json.SyntaxError); ok {
		offset = int(e.Offset)
	}
	linestr, line, column := gojq.GetLineByOffset(err.contents, offset)
	if line += err.line; line > 1 {
		return fmt.Sprintf("invalid json: %s:%d\n%s  %s",
			err.fname, line, gojq.FormatLineInfo(linestr, line, column), err.err)
	}
	return fmt.Sprintf("invalid json: %s\n    %s\n    %*c  %s",
		err.fname, linestr, column+1, '^', err.err)
}

type yamlParseError struct {
	fname, contents string
	err             error
}

func (err *yamlParseError) Error() string {
	var line int
	msg := strings.TrimPrefix(
		strings.TrimPrefix(err.err.Error(), "yaml: "),
		"unmarshal errors:\n  ")
	if fmt.Sscanf(msg, "line %d: ", &line); line == 0 {
		return "invalid yaml: " + err.fname
	}
	msg = msg[strings.Index(msg, ": ")+2:]
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = msg[:i]
	}
	linestr := gojq.GetLineByLine(err.contents, line)
	return fmt.Sprintf("invalid yaml: %s:%d\n%s  %s",
		err.fname, line, gojq.FormatLineInfo(linestr, line, 0), msg)
}

