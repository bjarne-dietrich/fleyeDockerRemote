package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func defaultServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "HELLO\n")
}

func stopContainers(cli *client.Client, ctx context.Context, imageName string) {
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		if imageName == container.Image {
			fmt.Printf("Found Container: %s with ID: %s", container.Image, container.ID)
			fmt.Println(container.Status)
			if !strings.Contains(container.Status, "Exited") {
				fmt.Print("Stopping container ", container.ID[:10], "... ")
				if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
					panic(err)
				}
				fmt.Println("Success")
			}
			if err := cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{}); err != nil {
				panic(err)
			}
			fmt.Printf("Deleted %s\n", container.ID)

		}
	}
}

func createRTSPPortMap(hostIP string) nat.PortMap {
	return nat.PortMap(
		map[nat.Port][]nat.PortBinding{
			nat.Port("8000/tcp"): {
				{
					HostIP:   hostIP,
					HostPort: getNewPort(),
				},
			},
			nat.Port("8001/tcp"): {
				{
					HostIP:   hostIP,
					HostPort: getNewPort(),
				},
			},
			nat.Port("8554/tcp"): {
				{
					HostIP:   hostIP,
					HostPort: getNewPort(),
				},
			},
		},
	)
}

func getNewPort() string {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	listener.Close()

	fmt.Printf("%d \n", listener.Addr().(*net.TCPAddr).Port)

	return fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
}

func deployRTSPContainer(cli *client.Client, ctx context.Context, hostIP string) container.ContainerCreateCreatedBody {
	out, err := cli.ImagePull(ctx, "aler9/rtsp-simple-server", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(os.Stdout, out)

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: "aler9/rtsp-simple-server",
		},
		&container.HostConfig{
			PortBindings: createRTSPPortMap(hostIP),
		}, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	return resp
}

func main() {
	imageName := "aler9/rtsp-simple-server"

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	hostIP := "0.0.0.0"
	stopContainers(cli, ctx, imageName)
	fmt.Println("Starting Service.")
	for i := 0; i < 5; i++ {
		resp := deployRTSPContainer(cli, ctx, hostIP)
		fmt.Println(resp.ID)
	}

	time.Sleep(3600 * time.Second)
}
