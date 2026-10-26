package engine

import (
	"context"
	"log/slog"

	"github.com/rm4n0s/glutter/internal/platform/fake"
)

// RunOnFake runs the engine on the fake platform with a renderer that draws
// nothing, for the tests of package engine_test that run the framework in the
// real loop (they cannot be in package engine: the framework imports it).
func RunOnFake(ctx context.Context, cfg Config, p *fake.Platform) error {
	cfg.connectors = map[string]func() (connection, error){"": func() (connection, error) { return &conn{Platform: p}, nil }}
	cfg.newRenderer = func(*Config, *slog.Logger) renderer { return &fakeRenderer{} }
	return Run(ctx, cfg)
}
