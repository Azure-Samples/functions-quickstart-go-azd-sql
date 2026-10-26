package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"functions-quickstart-go-azd-sql/internal/database"
)

func main() {
	operation := flag.String("operation", "insert", "insert, update, or delete")
	id := flag.Int("id", 0, "ProductId for update/delete")
	name := flag.String("name", "Coffee", "product name")
	cost := flag.Int("cost", 250, "cost in cents")
	flag.Parse()
	if err := run(*operation, *id, *name, *cost); err != nil {
		log.Fatal(err)
	}
}

func run(operation string, id int, name string, cost int) error {
	if operation != "insert" && operation != "update" && operation != "delete" {
		return fmt.Errorf("operation must be insert, update, or delete")
	}
	if operation != "insert" && id <= 0 {
		return fmt.Errorf("set -id to a positive ProductId")
	}
	if operation != "delete" && (name == "" || cost < 0) {
		return fmt.Errorf("name is required and cost must be non-negative")
	}
	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	switch operation {
	case "insert":
		err = db.QueryRowContext(ctx,
			"INSERT dbo.Products (Name, Cost) OUTPUT INSERTED.ProductId VALUES (@name, @cost)",
			sql.Named("name", name), sql.Named("cost", cost),
		).Scan(&id)
	default:
		query := "DELETE FROM dbo.Products WHERE ProductId = @id"
		args := []any{sql.Named("id", id)}
		if operation == "update" {
			query = "UPDATE dbo.Products SET Name = @name, Cost = @cost WHERE ProductId = @id"
			args = append(args, sql.Named("name", name), sql.Named("cost", cost))
		}
		var result sql.Result
		result, err = db.ExecContext(ctx, query, args...)
		if err == nil {
			var affected int64
			affected, err = result.RowsAffected()
			if err == nil && affected != 1 {
				err = fmt.Errorf("product %d was not found", id)
			}
		}
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s ProductId=%d\n", operation, id)
	return nil
}
