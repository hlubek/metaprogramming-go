package domain

import "github.com/gofrs/uuid"

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
