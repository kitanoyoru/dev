package docker

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"
	typesimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/samber/lo"
)

type Engine struct {
	client *client.Client
	cfg    *Config
}

func New(cfg *Config) (*Engine, error) {
	opts := []client.Opt{client.FromEnv}
	if cfg != nil && cfg.Host != "" {
		opts = append(opts, client.WithHost(cfg.Host))
	}
	if cfg == nil || cfg.APIVersionNegotiation {
		opts = append(opts, client.WithAPIVersionNegotiation())
	}

	c, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	return &Engine{
		client: c,
		cfg:    cfg,
	}, nil
}

func (d *Engine) Create(ctx context.Context, image string) (*string, error) {
	pullPolicy := strings.ToLower(d.cfg.PullPolicy)
	if pullPolicy == "always" || pullPolicy == "ifnotpresent" || pullPolicy == "if-not-present" {
		if err := d.ensureImage(ctx, image, pullPolicy); err != nil {
			return nil, err
		}
	}

	containerCfg, hostCfg, netCfg, name, err := d.buildContainerConfigs(image)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.ContainerCreate(ctx, containerCfg, hostCfg, netCfg, nil, name)
	if err != nil {
		return nil, err
	}

	if err := d.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

	return &resp.ID, nil
}

func (d *Engine) Remove(ctx context.Context, id string) error {
	return d.client.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func (d *Engine) ensureImage(ctx context.Context, image, policy string) error {
	if policy == "always" {
		rc, err := d.client.ImagePull(ctx, image, typesimage.PullOptions{})
		if err != nil {
			return err
		}
		defer rc.Close()
		_, _ = io.Copy(io.Discard, rc)
		return nil
	}

	imgs, err := d.client.ImageList(ctx, typesimage.ListOptions{})
	if err != nil {
		return err
	}
	if lo.ContainsBy(imgs, func(img typesimage.Summary) bool {
		return lo.Contains(img.RepoTags, image)
	}) {
		return nil
	}
	rc, err := d.client.ImagePull(ctx, image, typesimage.PullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	_, _ = io.Copy(io.Discard, rc)
	return nil
}

func (d *Engine) buildContainerConfigs(image string) (*container.Config, *container.HostConfig, *network.NetworkingConfig, string, error) {
	cc := &container.Config{Image: image}
	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}
	name := ""

	if d.cfg == nil {
		return cc, hc, nc, name, nil
	}

	defaults := d.cfg.DefaultContainer

	if len(defaults.Env) > 0 {
		envs := make([]string, 0, len(defaults.Env))
		// stable order for reproducibility
		keys := make([]string, 0, len(defaults.Env))
		for k := range defaults.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			envs = append(envs, fmt.Sprintf("%s=%s", k, defaults.Env[k]))
		}
		cc.Env = envs
	}
	if len(defaults.Cmd) > 0 {
		cc.Cmd = defaults.Cmd
	}
	if len(defaults.Labels) > 0 {
		cc.Labels = defaults.Labels
	}

	if len(defaults.Ports) > 0 {
		portSet := nat.PortSet{}
		portMap := nat.PortMap{}
		for containerPort, hostPort := range defaults.Ports {
			// containerPort may include protocol, e.g. "80/tcp". If not, assume tcp
			proto := "tcp"
			cp := strings.TrimSpace(containerPort)
			if strings.Contains(cp, "/") {
				parts := strings.SplitN(cp, "/", 2)
				cp = parts[0]
				proto = parts[1]
			}
			p, err := nat.NewPort(proto, cp)
			if err != nil {
				continue
			}
			portSet[p] = struct{}{}
			if strings.TrimSpace(hostPort) != "" {
				portMap[p] = []nat.PortBinding{{HostPort: strings.TrimSpace(hostPort)}}
			}
		}
		cc.ExposedPorts = portSet
		hc.PortBindings = portMap
	}

	if len(defaults.Volumes) > 0 {
		var binds []string
		// stable order
		keys := make([]string, 0, len(defaults.Volumes))
		for host := range defaults.Volumes {
			keys = append(keys, host)
		}
		sort.Strings(keys)
		for _, host := range keys {
			containerPath := defaults.Volumes[host]
			if containerPath == "" {
				continue
			}
			binds = append(binds, fmt.Sprintf("%s:%s", strings.TrimSpace(host), strings.TrimSpace(containerPath)))
		}
		hc.Binds = binds
	}

	hc.AutoRemove = defaults.AutoRemove
	if defaults.RestartPolicy != "" {
		hc.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(defaults.RestartPolicy)}
	}
	hc.Resources = container.Resources{
		Memory:    defaults.Resources.MemoryBytes,
		NanoCPUs:  defaults.Resources.NanoCPUs,
		CPUPeriod: defaults.Resources.CPUPeriod,
		CPUQuota:  defaults.Resources.CPUQuota,
	}

	if defaults.Network != "" {
		nc.EndpointsConfig = map[string]*network.EndpointSettings{
			defaults.Network: {},
		}
	}
	name = strings.TrimSpace(defaults.Name)

	return cc, hc, nc, name, nil
}
