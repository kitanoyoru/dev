package engine

import (
	"context"

	enginedocker "github.com/kitanoyoru/dev/engine/docker"
)

type Engine interface {
	Create(ctx context.Context, image string) (string, error)
	Remove(ctx context.Context, id string) error
	Update(ctx context.Context, id string) error
}

func New(config Config) (Engine, error) {
	switch config.Type {
	case "docker":
		return enginedocker.New(config.Docker, config.LogsWriter)
	default:
		return nil, nil
	}
}
