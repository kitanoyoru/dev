package docker

type (
	Config struct {
		Host                  string
		APIVersionNegotiation bool
		PullPolicy            string
		DefaultContainer      ContainerConfig
	}

	ContainerConfig struct {
		Name          string
		Env           map[string]string
		Cmd           []string
		Labels        map[string]string
		Ports         map[string]string
		Volumes       map[string]string
		AutoRemove    bool
		RestartPolicy string
		Network       string
		Resources     ResourceLimits
	}

	ResourceLimits struct {
		MemoryBytes int64
		NanoCPUs    int64
		CPUPeriod   int64
		CPUQuota    int64
	}
)

func (cfg *Config) Type() string {
	return "docker"
}
