package domain

import "github.com/gofrs/uuid"

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
