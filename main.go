package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/azure/azure-functions-golang-worker/sdk"
	"github.com/azure/azure-functions-golang-worker/sdk/bindings"
	"github.com/azure/azure-functions-golang-worker/worker"
)

type Product struct {
	ProductID int    `json:"ProductId"`
	Name      string `json:"Name"`
	Cost      int    `json:"Cost"`
}

func main() {
	worker.Start(newFunctionApp())
}

func newFunctionApp() *sdk.App {
	app := sdk.FunctionApp()
	app.SQL("productsChanged", productsChanged,
		sdk.WithTable("dbo.Products"),
		sdk.WithConnection("SqlConnectionString"),
	)
	return app
}

func productsChanged(ctx context.Context, changes []bindings.SQLChange) error {
	for _, change := range changes {
		var product Product
		if err := json.Unmarshal(change.Item, &product); err != nil {
			return fmt.Errorf("decode %s row: %w", change.Operation, err)
		}
		if product.ProductID <= 0 {
			return fmt.Errorf("ProductId must be positive")
		}
		switch change.Operation {
		case bindings.SQLOperationInsert, bindings.SQLOperationUpdate:
			if product.Name == "" || product.Cost < 0 {
				return fmt.Errorf("product %d requires Name and non-negative Cost", product.ProductID)
			}
		case bindings.SQLOperationDelete:
			// Deleted rows contain primary-key values only.
		default:
			return fmt.Errorf("unsupported SQL operation: %s", change.Operation)
		}
		slog.InfoContext(ctx, "product change processed",
			"operation", change.Operation.String(),
			"product_id", product.ProductID,
			"name", product.Name,
			"cost", product.Cost,
		)
	}
	return nil
}
