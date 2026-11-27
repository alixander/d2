// Run inside a checkout of d2lang/d2. Read a diagram from stdin and print its edited source.
package main

import (
	"fmt"
	"os"

	"github.com/d2lang/d2/d2compiler"
	"github.com/d2lang/d2/d2format"
	"github.com/d2lang/d2/d2oracle"
)

func main() {
	g, _, err := d2compiler.Compile("input.d2", os.Stdin, nil)
	if err != nil {
		panic(err)
	}
	switch {
	case len(os.Args) == 4 && os.Args[1] == "set":
		g, err = d2oracle.Set(g, nil, os.Args[2], nil, &os.Args[3])
	case len(os.Args) == 3 && os.Args[1] == "delete":
		g, err = d2oracle.Delete(g, nil, os.Args[2])
	default:
		fmt.Fprintln(os.Stderr, "usage: editor set KEY VALUE | editor delete KEY")
		os.Exit(2)
	}
	if err != nil {
		panic(err)
	}
	fmt.Print(d2format.Format(g.AST))
}
