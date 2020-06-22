# Metaprogramming with Go - or how to build code generators that parse Go code

The ideal Go code is optimized for legibility (easy to read, follow and understand). It favors explicitness over cleverness and tricks to save some lines of code. I think that's one of my favorite "features" of the Go language / mindset. It allows me to jump straight into the code of many popular projects. While something like the Kubernetes codebase is not exactly small and can be quite intimidating to get started with, it's nonetheless mostly functions calling other functions. Any editor offering code assistance for Go makes it easy to just follow calls and understand the flow of logic inside a program.

So while Go certainly has some nice features, it's also pretty minimal (by design). There are no generics (yet), but that will most probably change soon. And not every solution can be designed in a way where most of the boilerplate (rather boring code that sets things up or converts between different layers) can be removed. 

## Example: abstractions for database access

We want to create an abstraction of `Product` database records as entities in a repository.

First the domain model as the entity:

```go
type Product struct {
	ID                uuid.UUID
	ArticleNumber     string
	Name              string
	Description       string
	Color             string
	Size              string
	StockAvailability int
	PriceCents        int
	OnSale            bool
}
```

A repository can be just a bunch of functions operating on a database (or transaction) and this type. I like to use [squirrel](https://github.com/Masterminds/squirrel) because it's not an ORM but just helps to write SQL and reduce boilerplate:

```go
func InsertProduct(ctx context.Context, runner squirrel.BaseRunner, product domain.Product) error {
	_, err := squirrel.Insert("products").
		SetMap(map[string]interface{}{
			"product_id": product.ID,
			"article_number": product.ArticleNumber,
			"name": product.Name,
			"description": product.Description,
			"color": product.Color,
			"size": product.Size,
			"stock_availability": product.StockAvailability,
			"price_cents": product.PriceCents,
			"on_sale": product.OnSale,
		}).
		ExecContext(ctx)
	return err
}
```

For updating we implement a `ChangeSet` type with optional (nillable) fields to select which columns to set or update:

```go
type ProductChangeSet struct {
	ArticleNumber     *string
	Name              *string
	Description       *string
	Color             *string
	Size              *string
	StockAvailability *int
	PriceCents        *int
	OnSale            *bool
}

func UpdateProduct(ctx context.Context, runner squirrel.BaseRunner, id uuid.UUID, changeSet ProductChangeSet) error {
	res, err := squirrel.
		Update("gamers").
		Where(squirrel.Eq{"gamer_id": id}).
		SetMap(changeSet.toMap()).
		RunWith(runner).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("executing update: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting affected rows: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("update affected %d rows, but expected exactly 1", rowsAffected)
	}
	return err
}
```

To get the map of fields that actually changed, we implement a `toMap` method on our `ProductChangeSet` type:

```go
func (c ProductChangeSet) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if c.ArticleNumber != nil {
		m["article_number"] = c.ArticleNumber
	}
	if c.Name != nil {
		m["name"] = c.Name
	}
	if c.Description != nil {
		m["description"] = c.Description
	}
	if c.Color != nil {
		m["color"] = c.Color
	}
	if c.Size != nil {
		m["size"] = c.Size
	}
	if c.StockAvailability != nil {
		m["stock_availability"] = c.StockAvailability
	}
	if c.PriceCents != nil {
		m["price_cents"] = c.PriceCents
	}
	if c.OnSale != nil {
		m["on_sale"] = c.OnSale
	}
	return m
}
```

So while now everything should be nicely typed for a consumer of the repository we ended up with quite some boilerplate code. Most fields are referenced at 4 different places and `ProductChangeSet` is almost a direct copy of `Product` - just with pointers to the original types for optional values. The column names are referenced at 2 different places and we haven't even yet implemented sorting or filtering. This makes adding new fields / columns a little too cumbersome and error-prone.

So can we do better without adding complexity by using an object relational mapper (ORM)?

## Using reflection for metaprogramming

Metaprogramming in its essence is about programming a program (haha, sooo meta isn't it). One of the ways is to add some additional metadata into struct tags, so there's only one source of truth:

```go
type Product struct {
	ID                uuid.UUID `col:"product_id"`
	ArticleNumber     string    `col:"article_number"`
	Name              string    `col:"name"`
	Description       string    `col:"description"`
	Color             string    `col:"color"`
	Size              string    `col:"size"`
	StockAvailability int       `col:"stock_availability"`
	PriceCents        int       `col:"price_cents"`
	OnSale            bool      `col:"on_sale"`
}
```

We could now leverage reflection, which is built-in into Go, and access those struct tags in at runtime:

```go
func InsertProductReflect(ctx context.Context, runner squirrel.BaseRunner, product domain.Product) error {
	m := make(map[string]interface{})
	t := reflect.TypeOf(product)
	for i:=0; i< t.NumField();i++ {
		field := t.Field(i)
		col := field.Tag.Get("col")
		if col != "" {
			m[col] = reflect.ValueOf(product).Field(i).Interface()
		}
	}

	_, err := squirrel.Insert("products").
		SetMap(m).
		RunWith(runner).
		ExecContext(ctx)
	return err
}
```

Note that this gets rid of some of the references to the column and field names, but it's now also quite unreadable and hard to follow. Well, we could extract that to a function that can be re-used across repository functions. But we now loose all information about field usage when going through the code. So we only made a trade-off between legibility and reducing repetition in the code.

And: we cannot get rid of our `ProductChangeSet` definition that copies many fields of `Product` unless we get rid of static typing all fields and just accept a map of `interface{}` values.

## Metaprogramming for real: writing code that generates code

So how can we go beyond pure Go without sacrificing the advantages of explicitness and code which is easy to follow?

In the Go toolstack there's a built-in command for generating code called `go generate`. It can be used to process special `//go:generate my-command args` comments in .go files and will invoke the specified command to generate code (which is up to the command). This way it's easy and uniform to include generated code into a project and it's quite common to do so in the Go world (maybe a consequence from the lack of generics in a way).

So before jumping into code again, let's think about the necessary steps to write the boilerplate code with a code generator:

1. Add metadata by using struct tags (✅)
2. Implement a code generator that parses our Go code, extracts type information with struct tags and generates a new type and methods (🤔)

Well, luckily the Go toolchain includes a tokenizer and parser for Go code. There's not too much information out there, but you can find some nice articles (like https://arslan.io/2017/09/14/the-ultimate-guide-to-writing-a-go-tool/) about using the parser and working with the Go AST.

When first sketching a generator for this exact problem, I just used the Go AST to extract struct tags, which works quite well after fiddeling with the data structure. This is a visualization of the AST using http://goast.yuroyoro.net/:

![Go AST visualization](https://dev-to-uploads.s3.amazonaws.com/i/lxpjlv0d2dwsppw8ldaa.png)

So you can see it's much more low-level than using `reflect` and closely resembles the structure of the Go language. By using `ast.Inspect(node, func(n ast.Node) bool { ... }` you can visit the nodes and check if it matches a top-level declaration and then inspect all the fields with tags etc.

Just turns out it's pretty hard to get type informations for field declarations in a reliable way. This is, because in the end a definition like `Color string` is just two identifiers `*ast.Ident`. For built-ins this is still easy. But what about `uuid.UUID` (which is a `*ast.SelectorExpr`)? How to get the full import path to write it correctly in the generated code and what if we used anonymous imports or are refering to types in the same or some other package?

Turns out this is a rather hard problem and luckily (again) there's a type checker package that's doing all the hard work called [go/types](https://go.googlesource.com/example/+/HEAD/gotypes). Too bad, it's not working with modules to resolve packages. But we don't have to worry, since there's [golang.org/x/tools/go/packages](https://godoc.org/golang.org/x/tools/go/packages) which does all that.

## Code Generator Part 1

Again, there's not too much information out there on how to get started with this, so here's how a first attempt at the code generator could look like (without actually generating code):

```go
package main

import (
	"fmt"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	// 1. Handle arguments to command
	if len(os.Args) != 2 {
		failErr(fmt.Errorf("expected exactly one argument: <source type>"))
	}
	sourceType := os.Args[1]
	sourceTypePackage, sourceTypeName := splitSourceType(sourceType)

	// 2. Inspect package and use type checker to infer imported types
	pkg := loadPackage(sourceTypePackage)

	// 3. Lookup the given source type name in the package declarations
	obj := pkg.Types.Scope().Lookup(sourceTypeName)
	if obj == nil {
		failErr(fmt.Errorf("%s not found in declared types of %s",
			sourceTypeName, pkg))
	}

	// 4. We check if it is a declared type
	if _, ok := obj.(*types.TypeName); !ok {
		failErr(fmt.Errorf("%v is not a named type", obj))
	}
	// 5. We expect the underlying type to be a struct
	structType, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		failErr(fmt.Errorf("type %v is not a struct", obj))
	}

	// 6. Now we can iterate through fields and access tags
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tagValue := structType.Tag(i)
		fmt.Println(field.Name(), tagValue, field.Type())
	}
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
```

So we now end up with some more code as in the version using `reflect`. What this program does is basically these steps:

1. Handle arguments
2. Load type information via `packages`
3. Make a lookup into the types of the package scope
4. Check if the type exists...
5. ...and is a struct
6. Iterate through struct fields and tags with declared type

How to run this generator?

Let's add a mostly empty `mapping.go` file in the `repository` package:

```go
//go:generate go run github.com/hlubek/metaprogramming-go/cmd/generator github.com/hlubek/metaprogramming-go/domain.Product
package repository
```

This is a stub for the above-mentioned `go generate` command. In a project set up with Go modules, you can now run `go generate ./...` in the module root and the generator will be called:

```
$ go generate ./...
ID col:"product_id" github.com/gofrs/uuid.UUID
ArticleNumber col:"article_number" string
Name col:"name" string
Description col:"description" string
Color col:"color" string
Size col:"size" string
StockAvailability col:"stock_availability" int
PriceCents col:"price_cents" int
OnSale col:"on_sale" bool
```

Note that tags are not yet parsed, so we have to deal with this ourselves. But types are now fully qualified, so we can use that information to generate a `ProductChangeSet` struct type.

## How to generate code?

So how can we now generate the needed Go code? Of course you could use string manipulation or use the Go `text/template` package (many tools actually do this).

I recently saw [jennifer](https://github.com/dave/jennifer) a nice tool that has a Go API to generate Go code and includes proper handling of qualified types, formatting and some more.

Turns out it's quite easy to use the information we now have at our hands and use `jennifer` to generate the code we previously coded by hand.

## Code Generator Part 2

```go
package main

import (
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen"
	"golang.org/x/tools/go/packages"
)

func main() {
	// ...

	// Generate code using jennifer
	err := generate(sourceTypeName, structType)
	if err != nil {
		failErr(err)
	}
}

func generate(sourceTypeName string, structType *types.Struct) error {

	// 1. Get the package of the file with go:generate comment
	goPackage := os.Getenv("GOPACKAGE")

	// 2. Start a new file in this package
	f := NewFile(goPackage)

	// 3. Add a package comment, so IDEs detect files as generated
	f.PackageComment("Code generated by generator, DO NOT EDIT.")

	var (
		changeSetFields []Code
	)

	// 4. Iterate over struct fields
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

	// 5. Generate changeset type
	changeSetName := sourceTypeName + "ChangeSet"
	f.Type().Id(changeSetName).Struct(changeSetFields...)

	// 6. Build the target file name
	goFile := os.Getenv("GOFILE")
	ext := filepath.Ext(goFile)
	baseFilename := goFile[0 : len(goFile)-len(ext)]
	targetFilename := baseFilename + "_" + strings.ToLower(sourceTypeName) + "_gen.go"

	// 7. Write generated file
	return f.Save(targetFilename)
}

// ...

```

That's actually not too much new code here! All code is generated by using functions from the `jen` package and finally writing to a filename we derive from the filename where the `go:generate` comment was declared.

So, what's the output if we run the command again?

```
go generate ./...
cat repository/mapping_product_gen.go
```

```go
// Code generated by generator, DO NOT EDIT.
package repository

import uuid "github.com/gofrs/uuid"

type ProductChangeSet struct {
	ID                *uuid.UUID
	ArticleNumber     *string
	Name              *string
	Description       *string
	Color             *string
	Size              *string
	StockAvailability *int
	PriceCents        *int
	OnSale            *bool
}
```

Looks pretty nice so far. Now let's generate the `toMap()` method that returns a map for all the changes:


```go
import (
	// ...
        "regexp"
	// ...
)

// ...


// Use a simple regexp pattern to match tag values
var structColPattern = regexp.MustCompile(`col:"([^"]+)"`)

func generate(sourceTypeName string, structType *types.Struct) error {
	// ...

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

	// ...
}

// ...

```

And run the generator again:


```
go generate ./...
cat repository/mapping_product_gen.go
```

```go
// Code generated by generator, DO NOT EDIT.
package repository

import uuid "github.com/gofrs/uuid"

type ProductChangeSet struct {
	ID                *uuid.UUID
	ArticleNumber     *string
	Name              *string
	Description       *string
	Color             *string
	Size              *string
	StockAvailability *int
	PriceCents        *int
	OnSale            *bool
}

func (c ProductChangeSet) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if c.ID != nil {
		m["product_id"] = *c.ID
	}
	if c.ArticleNumber != nil {
		m["article_number"] = *c.ArticleNumber
	}
	if c.Name != nil {
		m["name"] = *c.Name
	}
	if c.Description != nil {
		m["description"] = *c.Description
	}
	if c.Color != nil {
		m["color"] = *c.Color
	}
	if c.Size != nil {
		m["size"] = *c.Size
	}
	if c.StockAvailability != nil {
		m["stock_availability"] = *c.StockAvailability
	}
	if c.PriceCents != nil {
		m["price_cents"] = *c.PriceCents
	}
	if c.OnSale != nil {
		m["on_sale"] = *c.OnSale
	}
	return m
}
```

This looks exactly like the code we previously wrote by hand! And the best thing: it still is readable and very explicit when inspecting what the code does. Only the generator itself is a little harder to follow and understand. But stepping through the generated code should be a breeze and not different from hand rolled code by any means.

## Conclusion

Writing code generators that uses existing Go code for metadata is not as hard as it might sound first. It's a viable solution to reduce boilerplate in many situations. When choosing the right tools the hard work of analyzing code and types as well as generating readable Go code is already done.

The code of this article can be found at https://github.com/hlubek/metaprogramming-go and is free to use under a MIT license.