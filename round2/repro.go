// Run from a d2lang/d2 checkout: go run /tmp/repro.go near|vars|wildcard
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/d2lang/d2/d2compiler"
	"github.com/d2lang/d2/d2format"
	"github.com/d2lang/d2/d2graph"
	"github.com/d2lang/d2/d2oracle"
)

const key = "x.style.fill-pattern.color"

func compile(source string) *d2graph.Graph {
	g, _, err := d2compiler.Compile("input.d2", strings.NewReader(source), nil)
	if err != nil {
		panic(err)
	}
	return g
}

func patternColor(g *d2graph.Graph, id string) string {
	obj, ok := g.Root.HasChild([]string{id})
	if !ok {
		return "<missing shape>"
	}
	if obj.Style.FillPatternColor == nil {
		return "<default>"
	}
	return obj.Style.FillPatternColor.Value
}

func main() {
	inputs := map[string]string{
		"near":     "x.style.fill-pattern.color: red\ny.near: x\n",
		"vars":     "vars: {swatch: {style.fill-pattern: dots {color: red}}}\nx: ${swatch}\ny: ${swatch}\n",
		"wildcard": "x\ny\n*.style.fill-pattern: dots {color: red}\n",
	}
	if len(os.Args) != 2 || inputs[os.Args[1]] == "" {
		fmt.Fprintln(os.Stderr, "usage: go run /tmp/repro.go near|vars|wildcard")
		os.Exit(2)
	}

	source := inputs[os.Args[1]]
	g := compile(source)
	fmt.Printf("Input\n%s\n", source)
	if os.Args[1] == "near" {
		fmt.Printf("Delete %s\nExpected source\nx\ny.near: x\n\n", key)
		edited, err := d2oracle.Delete(g, nil, key)
		if err != nil {
			fmt.Printf("Actual error\n%v\nCaller AST after the error\n%s", err, d2format.Format(g.AST))
			return
		}
		fmt.Printf("Actual source\n%s", d2format.Format(edited.AST))
		return
	}

	value := "white"
	fmt.Printf("Set %s to %s\nExpected colors: x=white, y=red\n", key, value)
	edited, err := d2oracle.Set(g, nil, key, nil, &value)
	if err != nil {
		fmt.Printf("Actual error\n%v\n", err)
		return
	}
	editedSource := d2format.Format(edited.AST)
	actual := compile(editedSource)
	fmt.Printf("Actual colors: x=%s, y=%s\nEdited source\n%s", patternColor(actual, "x"), patternColor(actual, "y"), editedSource)
}
