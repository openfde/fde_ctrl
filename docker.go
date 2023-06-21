package main

import (
	"context"
	"errors"
	"fde_ctrl/logger"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/strslice"
	"github.com/docker/go-connections/nat"
)

func startAndroidContainer(ctx context.Context, image, hostIP string) error {
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.41", nil, nil)
	if err != nil {
		return err
	}

	//image will store along with the installation
	// _, err = cli.ImagePull(ctx, "alpine", types.ImagePullOptions{})
	// if err != nil {
	// 	panic(err)

	// }
	shoudStartContainer := false
	containerID := ""
	containers, err := findAndoridContainers(ctx, cli)
	if err != nil {
		return err
	}
	if len(containers) > 0 {
		androidContainer := containers[0]
		//if container not running
		if androidContainer.State != "running" {
			shoudStartContainer = true
			containerID = androidContainer.ID
		}
	} else {
		containerConfig, hostConfig := constructAndroidContainerConfig(image, hostIP)
		resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, FDEContainerName)
		if err != nil {
			return err
		}
		shoudStartContainer = true
		containerID = resp.ID
	}
	if shoudStartContainer && len(containerID) > 0 {
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
		Binds:      []string{"/home/warlice/data:/data/data/com.termux/files/usr/data"},
		PortBindings: nat.PortMap{"5555": []nat.PortBinding{
			{
				HostIP:   "localhost",
				HostPort: "5555",
			},
		},
		},
	}
	exposedPort := make(map[nat.Port]struct{})
	exposedPort["5555"] = struct{}{}
	volumes := make(map[string]struct{})
	volumes["/data/data/com.termux/files/usr/data"] = struct{}{}
	containerConfig := &container.Config{
		Labels:       map[string]string{"os_version": "android"},
		ExposedPorts: exposedPort,
		Volumes:      volumes,
		ArgsEscaped:  false,
		Cmd: strslice.StrSlice{"androidboot.redroid_width=1920", "androidboot.redroid_height=1080",
			"android.redroid_dpi=480", "ro.host_ip=" + hostIP, "androidboot.redroid_net_ndns=114.114.114.114"},
		Image: image,
	}
	return containerConfig, hostConfig
}

func findAndoridContainers(ctx context.Context, cli *client.Client) ([]types.Container, error) {
	args := filters.NewArgs()
	args.Add("name", FDEContainerName)
	containerListOption := types.ContainerListOptions{
		All:    true,
		Filter: args,
	}

	return cli.ContainerList(ctx, containerListOption)
}

func waitContainerRunning(ctx context.Context, cli *client.Client) error {
	wait30Timer := time.NewTimer(time.Second * 30)
	runningChan := make(chan bool)
	go func(rc chan bool) {
		for {
			time.Sleep(time.Second * 1)
			containers, err := findAndoridContainers(ctx, cli)
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
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.41", nil, nil)
	if err != nil {
		return err
	}
	containers, err := findAndoridContainers(ctx, cli)
	if err != nil {
		return err
	}
	duration := time.Duration(time.Second * 30)
	for _, value := range containers {
		logger.Info("traversal_containers", value)
		logger.Info("traversal_containers_name", value.Names)
		if value.Command == "/init.kmre" && value.State == "running" {
			cli.ContainerStop(ctx, value.ID, &duration)
		}
	}
	return nil
}
