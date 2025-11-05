package engine

import (
	"context"

	"github.com/kitanoyoru/dev/engine/docker"
)

type Engine interface {
	Create(ctx context.Context, image string) (*string, error)
	Remove(ctx context.Context, id string) error
}

func Docker(config *docker.Config) (Engine, error) {
	return docker.New(config)
}
