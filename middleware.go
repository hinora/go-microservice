package goservice

// MiddlewareLocalActionFunc is the signature for a local-action middleware.
// action is the action being invoked; next is the rest of the pipeline (subsequent
// middlewares followed by the actual handler). Return an error to abort the call.
type MiddlewareLocalActionFunc func(ctx *Context, action Action, next func(*Context) (interface{}, error)) (interface{}, error)

// Middleware groups optional hook functions that intercept broker operations.
// Only LocalAction is currently supported; additional hooks (e.g. RemoteAction, Emit)
// may be added here in the future.
type Middleware struct {
	// LocalAction wraps the execution of every local action handler on this broker.
	// It is called after parameter validation but before Before hooks.
	LocalAction MiddlewareLocalActionFunc
}

// applyLocalActionMiddlewares wraps baseHandler with all registered LocalAction middlewares.
// Middlewares are applied so that the first registered middleware is the outermost wrapper.
func (b *Broker) applyLocalActionMiddlewares(ctx *Context, action Action, baseHandler func(*Context) (interface{}, error)) func(*Context) (interface{}, error) {
	h := baseHandler
	for i := len(b.Config.Middlewares) - 1; i >= 0; i-- {
		m := b.Config.Middlewares[i]
		if m.LocalAction == nil {
			continue
		}
		outerFn := m.LocalAction
		innerH := h
		h = func(c *Context) (interface{}, error) {
			return outerFn(c, action, innerH)
		}
	}
	return h
}
