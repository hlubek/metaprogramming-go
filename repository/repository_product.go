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