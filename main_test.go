package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/azure/azure-functions-golang-worker/sdk"
	"github.com/azure/azure-functions-golang-worker/sdk/bindings"
)

func TestProductsChanged(t *testing.T) {
	for _, tc := range []struct {
		name      string
		operation bindings.SQLOperation
		item      string
		wantError bool
	}{
		{"insert", bindings.SQLOperationInsert, `{"ProductId":1,"Name":"Coffee","Cost":250}`, false},
		{"update free product", bindings.SQLOperationUpdate, `{"ProductId":1,"Name":"Coffee","Cost":0}`, false},
		{"delete primary key only", bindings.SQLOperationDelete, `{"ProductId":1}`, false},
		{"missing primary key", bindings.SQLOperationDelete, `{}`, true},
		{"missing name", bindings.SQLOperationInsert, `{"ProductId":1,"Cost":250}`, true},
		{"negative cost", bindings.SQLOperationUpdate, `{"ProductId":1,"Name":"Coffee","Cost":-1}`, true},
		{"invalid JSON", bindings.SQLOperationInsert, `{`, true},
		{"null row", bindings.SQLOperationInsert, `null`, true},
		{"unknown operation", bindings.SQLOperation(99), `{"ProductId":1}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := productsChanged(context.Background(), []bindings.SQLChange{
				{Operation: tc.operation, Item: json.RawMessage(tc.item)},
			})
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
		})
	}
}

func TestProductsChangedBatch(t *testing.T) {
	if err := productsChanged(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	changes := []bindings.SQLChange{
		{Operation: bindings.SQLOperationInsert, Item: json.RawMessage(`{"ProductId":1,"Name":"Coffee","Cost":250}`)},
		{Operation: bindings.SQLOperationDelete, Item: json.RawMessage(`{"ProductId":2}`)},
	}
	if err := productsChanged(context.Background(), changes); err != nil {
		t.Fatal(err)
	}
	changes[1].Item = json.RawMessage(`{`)
	if err := productsChanged(context.Background(), changes); err == nil {
		t.Fatal("expected malformed row to fail the batch")
	}
}

func TestSQLRegistration(t *testing.T) {
	count := 0
	newFunctionApp().GetRegisteredFunctions().Range(func(_, value any) bool {
		count++
		fn := value.(*sdk.RegisteredFunction)
		binding := fn.TriggerBinding()
		if fn.FuncName != "productsChanged" || binding == nil || binding.SQLBinding == nil {
			t.Fatal("missing SQL trigger")
		}
		b := binding.SQLBinding
		if b.TableName != "dbo.Products" || b.ConnectionStringSetting != "SqlConnectionString" {
			t.Fatalf("unexpected SQL configuration: %+v", b)
		}
		return true
	})
	if count != 1 {
		t.Fatalf("registered functions = %d, want 1", count)
	}
}
