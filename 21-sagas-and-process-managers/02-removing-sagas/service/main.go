package service

import (
	"context"
	"remove_sagas/common"
	"remove_sagas/orders"
	"remove_sagas/stock"
)

func Run(ctx context.Context) {
	common.StartService(
		ctx,
		[]common.AddHandlersFn{
			orders.Initialize,
			stock.Initialize,
		},
	)
}
