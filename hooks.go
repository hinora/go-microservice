package goservice

// BeforeHookFunc is called before an action handler executes.
// Returning a non-nil error aborts the action and propagates the error to the caller.
type BeforeHookFunc func(ctx *Context) error

// AfterHookFunc is called after an action handler executes successfully.
// It may inspect or transform the result before it is returned.
type AfterHookFunc func(ctx *Context, result interface{}) (interface{}, error)

// ErrorHookFunc is called when an action handler returns an error.
// It may recover from the error by returning nil, or replace / wrap the error.
type ErrorHookFunc func(ctx *Context, err error) error

// ActionHooks holds lifecycle hooks for an action or a service (service-wide hooks apply to all actions).
type ActionHooks struct {
	Before []BeforeHookFunc
	After  []AfterHookFunc
	Error  []ErrorHookFunc
}

// runBeforeHooks executes service-level and then action-level Before hooks in order.
// The first hook that returns a non-nil error stops execution and returns that error.
func (b *Broker) runBeforeHooks(ctx *Context, action Action) error {
	if ctx.Service != nil {
		for _, h := range ctx.Service.Hooks.Before {
			if err := h(ctx); err != nil {
				return err
			}
		}
	}
	for _, h := range action.Hooks.Before {
		if err := h(ctx); err != nil {
			return err
		}
	}
	return nil
}

// runAfterHooks executes service-level and then action-level After hooks in order.
// Each hook may transform the result; the first hook that returns an error stops the chain.
func (b *Broker) runAfterHooks(ctx *Context, action Action, result interface{}) (interface{}, error) {
	if ctx.Service != nil {
		for _, h := range ctx.Service.Hooks.After {
			var err error
			result, err = h(ctx, result)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, h := range action.Hooks.After {
		var err error
		result, err = h(ctx, result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// runErrorHooks executes service-level and then action-level Error hooks in order.
// A hook that returns nil is treated as a recovery — the chain stops and nil is returned.
func (b *Broker) runErrorHooks(ctx *Context, action Action, err error) error {
	if ctx.Service != nil {
		for _, h := range ctx.Service.Hooks.Error {
			err = h(ctx, err)
			if err == nil {
				return nil
			}
		}
	}
	for _, h := range action.Hooks.Error {
		err = h(ctx, err)
		if err == nil {
			return nil
		}
	}
	return err
}
