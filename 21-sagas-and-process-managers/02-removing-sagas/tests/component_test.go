// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"remove_sagas/service"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		service.Run(ctx)
	}()

	waitForHttpServer(t)

	product1ID := uuid.New().String()
	product1Quantity := 2
	addProductToStock(t, product1ID, product1Quantity)

	product2ID := uuid.New().String()
	product2Quantity := 3
	addProductToStock(t, product2ID, product2Quantity)

	t.Run("order_shipped", func(t *testing.T) {
		orderID := uuid.New().String()
		placeOrder(t, orderID, map[string]int{
			product1ID: 1,
			product2ID: 1,
		})
		requireOrderShipped(t, orderID)
	})

	t.Run("out_of_order_product", func(t *testing.T) {
		orderID := uuid.New().String()
		placeOrder(t, orderID, map[string]int{
			product1ID: 1,
			product2ID: 3, // 1 should be missing
		})
		requireOrderCancelled(t, orderID)
	})

	t.Run("order_shipped_with_all_left_products", func(t *testing.T) {
		orderID := uuid.New().String()
		placeOrder(t, orderID, map[string]int{
			product1ID: 1,
			product2ID: 2,
		})
		requireOrderShipped(t, orderID)
	})
}

func requireOrderShipped(t *testing.T, orderID string) {
	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			order := getOrder(t, orderID)
			if !assert.NotEmpty(t, order) {
				return
			}

			if !assert.True(t, order.Shipped) {
				return
			}

			if !assert.False(t, order.Cancelled) {
				return
			}
		},
		time.Second*5,
		time.Millisecond*200,
	)
}

func requireOrderCancelled(t *testing.T, orderID string) {
	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			order := getOrder(t, orderID)
			if !assert.NotEmpty(t, order) {
				return
			}

			if !assert.False(t, order.Shipped) {
				return
			}

			if !assert.True(t, order.Cancelled) {
				return
			}
		},
		time.Second*5,
		time.Millisecond*200,
	)
}

func waitForHttpServer(t *testing.T) {
	t.Helper()

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			resp, err := http.Get("http://localhost:8080/health")
			if !assert.NoError(t, err) {
				return
			}
			defer resp.Body.Close()

			if assert.Less(t, resp.StatusCode, 300, "API not ready, http status: %d", resp.StatusCode) {
				return
			}
		},
		time.Second*10,
		time.Millisecond*50,
	)
}

func addProductToStock(t *testing.T, productID string, quantity int) {
	t.Helper()

	reqBody := map[string]any{
		"product_id": productID,
		"quantity":   quantity,
	}

	resp, err := http.Post(
		"http://localhost:8080/products-stock",
		"application/json",
		bytes.NewReader(lo.Must(json.Marshal(reqBody))),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func placeOrder(t *testing.T, orderID string, products map[string]int) {
	t.Helper()

	reqBody := map[string]any{
		"order_id": orderID,
		"products": products,
	}

	resp, err := http.Post(
		"http://localhost:8080/orders",
		"application/json",
		bytes.NewReader(lo.Must(json.Marshal(reqBody))),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

type OrderResponse struct {
	OrderID   string `json:"order_id"`
	Shipped   bool   `json:"shipped"`
	Cancelled bool   `json:"cancelled"`
}

func getOrder(t assert.TestingT, orderID string) OrderResponse {
	resp, err := http.Get(
		"http://localhost:8080/orders/" + orderID,
	)
	if !assert.NoError(t, err) {
		return OrderResponse{}
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var orderResponse OrderResponse

	err = json.NewDecoder(resp.Body).Decode(&orderResponse)
	if !assert.NoError(t, err) {
		return OrderResponse{}
	}

	return orderResponse
}
