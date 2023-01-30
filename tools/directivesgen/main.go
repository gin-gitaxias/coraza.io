// Copyright 2023 The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"html/template"
	"log"
	"os"
	"strings"
	"time"
)

type Directive struct {
	Name             string
	Description      string
	Syntax           string
	Default          string
	Date             string
	LastModification string
	Content          string
}

//go:embed template.md
var contentTemplate string

const dstDir = "./content/docs/seclang/directives"

func main() {
	tmpl, err := template.New("directive").Parse(contentTemplate)
	if err != nil {
		log.Fatal(err)
	}

	src, err := os.ReadFile("./coraza/internal/seclang/directives.go")
	if err != nil {
		log.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "directives.go", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch fn := n.(type) {

		// catching all function declarations
		// other intersting things to catch FuncLit and FuncType
		case *ast.FuncDecl:
			fnName := fn.Name.String()
			if !strings.HasPrefix(fnName, "directive") {
				return true
			}

			if fn.Doc == nil {
				return true
			}

			directiveName := fnName[9:]
			f, err := os.Create(fmt.Sprintf("%s/%s.md", dstDir, directiveName))
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			d := parseDirective(directiveName, fn.Doc.Text())

			content := bytes.Buffer{}
			err = tmpl.Execute(&content, d)
			if err != nil {
				log.Fatal(err)
			}

			_, err = f.WriteString(html.UnescapeString(content.String()))
			if err != nil {
				log.Fatal(err)
			}
		}
		return true
	})
}

func parseDirective(name string, doc string) Directive {
	d := Directive{
		Name:             name,
		LastModification: time.Now().Format(time.RFC3339),
	}

	fieldAppenders := map[string]func(d *Directive, value string){
		"Description": func(d *Directive, value string) { d.Description += value },
		"Syntax":      func(d *Directive, value string) { d.Syntax += value },
		"Default":     func(d *Directive, value string) { d.Default += value },
	}

	previousKey := ""
	scanner := bufio.NewScanner(strings.NewReader(doc))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "directive") {
			continue
		}

		if len(strings.TrimSpace(scanner.Text())) == 0 {
			continue
		}

		if strings.HasPrefix(scanner.Text(), "---") {
			break
		}

		key, value, ok := strings.Cut(scanner.Text(), ": ")
		if !ok {
			key = previousKey
			value = " " + scanner.Text()
		}

		if fn, ok := fieldAppenders[key]; ok {
			fn(&d, value)
			previousKey = key
		} else if previousKey != "" {
			fieldAppenders[previousKey](&d, value)
		} else {
			log.Fatalf("unknown field %q", key)
		}
	}

	for scanner.Scan() {
		d.Content += decorateNote(scanner.Text()) + "\n"
	}

	return d
}

func decorateNote(s string) string {
	return strings.Replace(s, "Note:", "**Note:**", -1)
}
