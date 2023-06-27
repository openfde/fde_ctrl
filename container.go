package main

import (
	"context"
	"errors"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/go-connections/nat"
)

const dockerSocket = "unix:///var/run/docker.sock"

func startAndroidContainer(ctx context.Context, image, hostIP string) error {
	cli, err := client.NewClient(dockerSocket, "v1.41", nil, nil)
	if err != nil {
		return err
	}

	//image will store along with the installation
	// _, err = cli.ImagePull(ctx, "alpine", types.ImagePullOptions{})
	// if err != nil {
	// 	panic(err)

	// }
	shouldStartContainer, shouldCreate := false, false
	containerID := ""
	containers, err := findAndoridContainers(ctx, cli, FDEContainerName)
	if err != nil {
		return err
	}
	if len(containers) > 0 {
		androidContainer := containers[0]
		containerID = androidContainer.ID
		if androidContainer.Image != image {
			err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
			if err != nil {
				return err
			}
			shouldCreate = true
			containerID = ""
		} else {
			// if container not running
			if androidContainer.State != "running" {
				shouldStartContainer = true
			}
		}
	}
	if shouldCreate {
		containerConfig, hostConfig := constructAndroidContainerConfig(image, hostIP)
		resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, FDEContainerName)
		if err != nil {
			return err
		}
		shouldStartContainer = true
		containerID = resp.ID
	}
	if shouldStartContainer && len(containerID) > 0 {
		if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
			return err
		}
	}
	err = waitContainerRunning(ctx, cli)
	if err != nil {
		return err
	}
	return nil
}

const (
	FDEContainerName = "fdedroid"
)

func constructAndroidContainerConfig(image, hostIP string) (*container.Config, *container.HostConfig) {
	hostConfig := &container.HostConfig{
		Privileged: true,
		Binds: []string{"/run/user/1000/anbox/sockets/qemu_pipe:/dev/qemu_pipe",
			"/run/user/1000/anbox/sockets/anbox_audio:/dev/anbox_audio:rw",
			"/run/user/1000/anbox/sockets/anbox_bridge:/dev/anbox_bridge:rw",
			"/run/user/1000/anbox/input/event0:/dev/input/event0:rw",
			"/run/user/1000/anbox/input/event1:/dev/input/event1:rw",
			"/run/user/1000/anbox/input/event2:/dev/input/event2:rw"},
		PortBindings: nat.PortMap{"5555": []nat.PortBinding{
			{
				HostIP:   "localhost",
				HostPort: "5555",
			},
		},
		},
	}

	volumes := make(map[string]struct{})
	volumes["/dev/qemu_pipe"] = struct{}{}
	volumes["/dev/anbox_audio"] = struct{}{}
	volumes["/dev/anbox_bridge"] = struct{}{}
	volumes["/dev/input/event0"] = struct{}{}
	volumes["/dev/input/event1"] = struct{}{}
	volumes["/dev/input/event2"] = struct{}{}
	exposedPort := make(map[nat.Port]struct{})
	exposedPort["5555"] = struct{}{}
	containerConfig := &container.Config{
		Labels:       map[string]string{"os_version": "android"},
		ExposedPorts: exposedPort,
		ArgsEscaped:  false,
		Volumes:      volumes,
		Image:        image,
	}
	return containerConfig, hostConfig
}

func findAndoridContainers(ctx context.Context, cli *client.Client, name string) ([]types.Container, error) {
	containerListOption := types.ContainerListOptions{
		All: true,
	}
	if len(name) != 0 {
		args := filters.NewArgs()
		args.Add("name", name)
		containerListOption.Filter = args
	}

	return cli.ContainerList(ctx, containerListOption)
}

func waitContainerRunning(ctx context.Context, cli *client.Client) error {
	wait30Timer := time.NewTimer(time.Second * 30)
	runningChan := make(chan bool)
	go func(rc chan bool) {
		for {
			time.Sleep(time.Second * 1)
			containers, err := findAndoridContainers(ctx, cli, FDEContainerName)
			if err != nil {
				runningChan <- false
			}
			if len(containers) > 0 {
				if containers[0].State == "running" {
					runningChan <- true
				}
			}
		}
	}(runningChan)
	select {
	case <-ctx.Done():
		{
			return errors.New("main process canceled")
		}
	case <-wait30Timer.C:
		{
			return errors.New("time out for starting container")
		}
	case result := <-runningChan:
		{
			if result {
				return nil
			}
			return errors.New("watting for container running error")
		}
	}
}

func stopAndroidContainer(ctx context.Context, name string) error {
	cli, err := client.NewClient(dockerSocket, "v1.41", nil, nil)
	if err != nil {
		return err
	}
	containers, err := findAndoridContainers(ctx, cli, name)
	if err != nil {
		return err
	}
	duration := time.Duration(time.Second * 30)
	for _, value := range containers {
		if value.State == "running" {
			cli.ContainerStop(ctx, value.ID, &duration)
		}
	}
	return nil
}
