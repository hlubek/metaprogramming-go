package main

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/hlubek/metaprogramming-go/domain"
	"github.com/hlubek/metaprogramming-go/repository"
)

func main() {
	ctx := context.Background()

	repository.InsertProduct(ctx, nil, domain.Product{
		ID:                uuid.Must(uuid.NewV4()),
		ArticleNumber:     "12345678",
		Name:              "Cheddar cheese",
		Description:       "Very cheesy.",
		Color:             "orange",
		Size:              "large",
		StockAvailability: 3,
		PriceCents:        469,
		OnSale:            false,
	})
}
