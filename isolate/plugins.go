package isolate

import (
	"fmt"

	"context"
)

// BoxConstructor is a type of a Box constructor
type BoxConstructor func(context.Context, BoxConfig) (Box, error)

var (
	plugins = map[string]BoxConstructor{}
)

// RegisterBox adds isolate plugin to the plugins list
func RegisterBox(name string, constructor BoxConstructor) {
	plugins[name] = constructor
}

// ConstructBox creates new Box
func ConstructBox(ctx context.Context, name string, cfg BoxConfig) (Box, error) {
	constructor, ok := plugins[name]
	if !ok {
		return nil, fmt.Errorf("isolation %s is not available", name)
	}

	return constructor(ctx, cfg)
}
