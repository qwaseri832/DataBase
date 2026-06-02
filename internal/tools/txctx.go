package tools

import "context"

type ctxKey struct{}

// WithTxID кладёт номер транзакции в контекст.
func WithTxID(parent context.Context, id int64) context.Context {
	return context.WithValue(parent, ctxKey{}, id)
}

// TxID достаёт номер транзакции из контекста.
func TxID(ctx context.Context) int64 {
	v, _ := ctx.Value(ctxKey{}).(int64)
	return v
}
