package main

import (
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/dave/jennifer/jen"
	"golang.org/x/tools/go/packages"
)

func main() {
	// Handle arguments to command
	if len(os.Args) != 2 {
		failErr(fmt.Errorf("expected exactly one argument: <source type>"))
	}
	sourceType := os.Args[1]
	sourceTypePackage, sourceTypeName := splitSourceType(sourceType)

	// Inspect package and use type checker to infer imported types
	pkg := loadPackage(sourceTypePackage)

	// Lookup the given source type name in the package declarations
	obj := pkg.Types.Scope().Lookup(sourceTypeName)
	if obj == nil {
		failErr(fmt.Errorf("%s not found in declared types of %s",
			sourceTypeName, pkg))
	}

	// We check if it is a declared type
	if _, ok := obj.(*types.TypeName); !ok {
		failErr(fmt.Errorf("%v is not a named type", obj))
	}
	// We expect the underlying type to be a struct
	structType, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		failErr(fmt.Errorf("type %v is not a struct", obj))
	}

	// Generate code using jennifer
	err := generate(sourceTypeName, structType)
	if err != nil {
		failErr(err)
	}
}

// Use a simple regexp pattern to match tag values
var structColPattern = regexp.MustCompile(`col:"([^"]+)"`)

func generate(sourceTypeName string, structType *types.Struct) error {

	// Get the package of the file with go:generate comment
	goPackage := os.Getenv("GOPACKAGE")

	// Start a new file in this package
	f := NewFile(goPackage)

	// Add a package comment, so IDEs detect files as generated
	f.PackageComment("Code generated by generator, DO NOT EDIT.")

	var (
		changeSetFields []Code
	)

	// Iterate over struct fields
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)

		// Generate code for each changeset field
		code := Id(field.Name())
		switch v := field.Type().(type) {
		case *types.Basic:
			code.Op("*").Id(v.String())
		case *types.Named:
			typeName := v.Obj()
			// Qual automatically imports packages
			code.Op("*").Qual(
				typeName.Pkg().Path(),
				typeName.Name(),
			)
		default:
			return fmt.Errorf("struct field type not hanled: %T", v)
		}
		changeSetFields = append(changeSetFields, code)
	}

	// Generate change set type
	changeSetName := sourceTypeName + "ChangeSet"
	f.Type().Id(changeSetName).Struct(changeSetFields...)

	// 1. Collect code in toMap() block
	var toMapBlock []Code

	// 2. Build "m := make(map[string]interface{})"
	toMapBlock = append(toMapBlock, Id("m").Op(":=").Make(Map(String()).Interface()))

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tagValue := structType.Tag(i)

		matches := structColPattern.FindStringSubmatch(tagValue)
		if matches == nil {
			continue
		}
		col := matches[1]

		// 3. Build "if c.Field != nil { m["col"] = *c.Field }"
		code := If(Id("c").Dot(field.Name()).Op("!=").Nil()).Block(
			Id("m").Index(Lit(col)).Op("=").Op("*").Id("c").Dot(field.Name()),
		)
		toMapBlock = append(toMapBlock, code)
	}

	// 4. Build return statement
	toMapBlock = append(toMapBlock, Return(Id("m")))

	// 5. Build toMap method
	f.Func().Params(
		Id("c").Id(changeSetName),
	).Id("toMap").Params().Map(String()).Interface().Block(
		toMapBlock...,
	)

	// Build the target file name
	goFile := os.Getenv("GOFILE")
	ext := filepath.Ext(goFile)
	baseFilename := goFile[0 : len(goFile)-len(ext)]
	targetFilename := baseFilename + "_" + strings.ToLower(sourceTypeName) + "_gen.go"

	// Write generated file
	return f.Save(targetFilename)
}

func loadPackage(path string) *packages.Package {
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedImports}
	pkgs, err := packages.Load(cfg, path)
	if err != nil {
		failErr(fmt.Errorf("loading packages for inspection: %v", err))
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	return pkgs[0]
}

func splitSourceType(sourceType string) (string, string) {
	idx := strings.LastIndexByte(sourceType, '.')
	if idx == -1 {
		failErr(fmt.Errorf(`expected qualified type as "pkg/path.MyType"`))
	}
	sourceTypePackage := sourceType[0:idx]
	sourceTypeName := sourceType[idx+1:]
	return sourceTypePackage, sourceTypeName
}

func failErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
