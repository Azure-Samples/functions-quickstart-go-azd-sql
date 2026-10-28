package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"functions-quickstart-go-azd-sql/internal/database"
)

//go:embed schema.sql
var schema string

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	name, clientID := os.Getenv("SQL_IDENTITY_NAME"), os.Getenv("SQL_IDENTITY_CLIENT_ID")
	if name == "" || clientID == "" {
		return fmt.Errorf("set SQL_IDENTITY_NAME and SQL_IDENTITY_CLIENT_ID from azd outputs")
	}
	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if _, err := db.ExecContext(ctx, schema,
		sql.Named("identityName", name),
		sql.Named("clientId", clientID),
	); err != nil {
		return fmt.Errorf("initialize SQL schema and permissions: %w", err)
	}
	fmt.Println("SQL change tracking and managed-identity permissions are ready.")
	return nil
}
