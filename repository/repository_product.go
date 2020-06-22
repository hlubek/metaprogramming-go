package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"github.com/hlubek/metaprogramming-go/domain"
)

func InsertProduct(ctx context.Context, runner squirrel.BaseRunner, product domain.Product) error {
	_, err := squirrel.Insert("products").
		SetMap(map[string]interface{}{
			"product_id":         product.ID,
			"article_number":     product.ArticleNumber,
			"name":               product.Name,
			"description":        product.Description,
			"color":              product.Color,
			"size":               product.Size,
			"stock_availability": product.StockAvailability,
			"price_cents":        product.PriceCents,
			"on_sale":            product.OnSale,
		}).
		RunWith(runner).
		ExecContext(ctx)
	return err
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
