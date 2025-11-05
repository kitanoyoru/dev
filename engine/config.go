package engine

import (
	"io"

	enginedocker "github.com/kitanoyoru/dev/engine/docker"
)

type Config struct {
	Type       string
	LogsWriter io.Writer
	Docker     *enginedocker.Config
}

type ConfigOption func(c *Config)

func WithLogsWriter(writer io.Writer) ConfigOption {
	return func(c *Config) {
		c.LogsWriter = writer
	}
}
