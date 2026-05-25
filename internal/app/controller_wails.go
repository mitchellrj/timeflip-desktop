package app

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (c *Controller) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	c.Startup(ctx)
	return nil
}

func (c *Controller) ServiceShutdown() error {
	c.Shutdown(context.Background())
	return nil
}
