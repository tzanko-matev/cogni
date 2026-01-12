package call

import "context"

// CallHook allows injecting behaviors around the model call.
type CallHook interface {
	BeforeCall(ctx context.Context, input CallInput) error
	AfterCall(ctx context.Context, input CallInput, result CallResult) error
}
