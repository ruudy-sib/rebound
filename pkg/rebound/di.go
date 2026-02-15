package rebound

import (
	"context"

	"go.uber.org/dig"
	"go.uber.org/zap"
)

// DIParams holds dependencies needed to create a Rebound instance via DI.
type DIParams struct {
	dig.In

	Logger *zap.Logger
	Config *Config `optional:"true"`
}

// ProvideRebound creates a Rebound instance for dependency injection.
// Use this when integrating Rebound into an app that uses uber-go/dig.
//
// Example:
//
//	container := dig.New()
//	container.Provide(rebound.ProvideRebound)
//	container.Invoke(func(rb *rebound.Rebound) {
//	    rb.Start(ctx)
//	})
func ProvideRebound(params DIParams) (*Rebound, error) {
	cfg := params.Config
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Use the provided logger
	cfg.Logger = params.Logger

	return New(cfg)
}

// RegisterWithContainer registers Rebound with a dig container.
// This is a convenience function that handles the registration for you.
//
// Example:
//
//	container := dig.New()
//	if err := rebound.RegisterWithContainer(container); err != nil {
//	    log.Fatal(err)
//	}
func RegisterWithContainer(container *dig.Container) error {
	return container.Provide(ProvideRebound)
}

// StartParams holds dependencies for starting Rebound via DI.
type StartParams struct {
	dig.In

	Rebound *Rebound
	Context context.Context `optional:"true"`
}

// StartRebound is a lifecycle hook that starts Rebound when invoked via DI.
//
// Example:
//
//	container.Invoke(rebound.StartRebound)
func StartRebound(params StartParams) error {
	ctx := params.Context
	if ctx == nil {
		ctx = context.Background()
	}

	return params.Rebound.Start(ctx)
}
