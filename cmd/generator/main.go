package main

import (
	"fmt"
	"go/types"
	"os"

	"golang.org/x/tools/go/packages"
)

func main() {
	// Special env variable set by "go generate"
	goFile := os.Getenv("GOFILE")

	if len(os.Args) != 2 {
		failErr(fmt.Errorf("expected exactly one argument: [source type]"))
	}

	sourceType := os.Args[1]

	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax}
	pkgs, err := packages.Load(cfg, fmt.Sprintf("file=%s", goFile))
	if err != nil {
		failErr(fmt.Errorf("loading packages for inspection: %v", err))
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	pkg := pkgs[0]

	obj := pkg.Types.Scope().Lookup(sourceType)
	if obj == nil {
		failErr(fmt.Errorf("%s not found in lookup", sourceType))
	}

	if _, ok := obj.(*types.TypeName); !ok {
		failErr(fmt.Errorf("%v is not a named type", obj))
	}
	structType, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		failErr(fmt.Errorf("type %v is a %T, not a struct", obj, obj.Type().Underlying()))
	}

	fmt.Println("Type name:", obj.Name(), "Type package:", obj.Pkg().Path())

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tagValue := structType.Tag(i)
		fmt.Println(field.Name(), tagValue, field.Type())
	}
}

func failErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
