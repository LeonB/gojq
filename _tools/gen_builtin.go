package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"regexp"
	"strings"

	"github.com/itchyny/astgen-go"
	"github.com/itchyny/gojq"
)

const fileFormat = `// Code generated by _tools/gen_builtin.go; DO NOT EDIT.

package gojq

func init() {%s}
`

func main() {
	var input, output string
	flag.StringVar(&input, "i", "", "input file")
	flag.StringVar(&output, "o", "", "output file")
	flag.Parse()
	if err := run(input, output); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input, output string) error {
	cnt, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	q, err := gojq.Parse(string(cnt))
	if err != nil {
		return err
	}
	fds := make(map[string][]*gojq.FuncDef)
	for _, fd := range q.FuncDefs {
		fd.Minify()
		fds[fd.Name] = append(fds[fd.Name], fd)
	}
	t, err := astgen.Build(fds)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("\n\tbuiltinFuncDefs = ")
	if err := printCompositeLit(&sb, t.(*ast.CompositeLit)); err != nil {
		return err
	}
	out := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	_, err = fmt.Fprintf(out, fileFormat, sb.String())
	return err
}

func printCompositeLit(out *strings.Builder, t *ast.CompositeLit) error {
	err := printer.Fprint(out, token.NewFileSet(), t.Type)
	if err != nil {
		return err
	}
	out.WriteString("{")
	for _, kv := range t.Elts {
		out.WriteString("\n\t\t")
		var sb strings.Builder
		err = printer.Fprint(&sb, token.NewFileSet(), kv)
		if err != nil {
			return err
		}
		str := sb.String()
		for op := gojq.OpPipe; op <= gojq.OpUpdateAlt; op++ {
			r := regexp.MustCompile(fmt.Sprintf(`\b((?:Update)?Op): %d\b`, op))
			str = r.ReplaceAllString(str, fmt.Sprintf("$1: %#v", op))
		}
		for termType := gojq.TermTypeIdentity; termType <= gojq.TermTypeQuery; termType++ {
			r := regexp.MustCompile(fmt.Sprintf(`(Term{Type): %d\b`, termType))
			str = r.ReplaceAllString(str, fmt.Sprintf("$1: %#v", termType))
		}
		out.WriteString(strings.ReplaceAll(str, "gojq.", ""))
		out.WriteString(",")
	}
	out.WriteString("\n\t}\n")
	return nil
}
