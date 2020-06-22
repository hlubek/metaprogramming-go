package main

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/hlubek/metaprogramming-go/repository"
)

func main() {
	ctx := context.Background()

	repository.UpdateProduct(ctx, nil, uuid.Must(uuid.FromString("b34081c7-9f33-4b04-ba33-3a112199f8c2")), repository.ProductChangeSet{
		ArticleNumber: stringPtr("12345678"),
		Name:          stringPtr("Cheddar cheese"),
	})
}

func stringPtr(s string) *string {
	return &s
}
