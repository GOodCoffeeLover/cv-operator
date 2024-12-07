package logger

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

// WithValues -- helper for add value to logger in context
func WithValues(ctx context.Context, keyValues interface{}) context.Context {
	return ctrl.LoggerInto(
		ctx,
		ctrl.LoggerFrom(ctx, keyValues),
	)
}
