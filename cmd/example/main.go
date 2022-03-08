package main

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gofrs/uuid"
	_ "github.com/lib/pq"

	"github.com/hlubek/metaprogramming-go/domain"
	"github.com/hlubek/metaprogramming-go/repository"
)

func main() {
	ctx := context.Background()

	err := run(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

//go:embed schema.sql
var schema []byte

func run(ctx context.Context) error {
	// Start an embedded PostgreSQL server

	var postgresLog bytes.Buffer

	postgres := embeddedpostgres.NewDatabase(
		embeddedpostgres.
			DefaultConfig().
			Version(embeddedpostgres.V13).
			Port(54329).
			Database("example").
			Logger(&postgresLog),
	)
	log.Printf("Starting embedded PostgreSQL server...")
	err := postgres.Start()
	if err != nil {
		return fmt.Errorf("starting embedded PostgreSQL server: %w", err)
	}

	defer func() {
		log.Printf("Stopping embedded PostgreSQL server")
		err = postgres.Stop()
		if err != nil {
			log.Printf("Failed to stop embedded PostgreSQL server: %v", err)
		}
	}()

	// Connect to the database

	connStr := "user=postgres password=postgres dbname=example host=localhost port=54329 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("opening database connection: %w", err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("executing schema: %w", err)
	}

	// Run some example repository operations:

	// Insert a product:

	productID := uuid.Must(uuid.FromString("b34081c7-9f33-4b04-ba33-3a112199f8c2"))

	err = repository.InsertProduct(ctx, db, domain.Product{
		ID: productID,
	})
	if err != nil {
		return fmt.Errorf("inserting product: %w", err)
	}
	log.Printf("Inserted product %s", productID)

	// Update an existing product with the generated change set:

	err = repository.UpdateProduct(ctx, db, productID, repository.ProductChangeSet{
		ArticleNumber: stringPtr("12345678"),
		Name:          stringPtr("Cheddar cheese"),
	})
	if err != nil {
		return fmt.Errorf("updating product: %w", err)
	}
	log.Printf("Updated product %s", productID)

	return nil
}

func stringPtr(s string) *string {
	return &s
}
