package runtime

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type FunctionRepository interface {
	GetFunction(ctx context.Context, path string) (io.Reader, error)
}

type Runtime struct {
	functionRepository FunctionRepository
	dockerClient       *client.Client
}

func NewRuntime(functionRepository FunctionRepository) (*Runtime, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Runtime{
		functionRepository: functionRepository,
		dockerClient:       dockerClient,
	}, nil
}

func (r *Runtime) RunFunction(ctx context.Context, path string) ([]byte, error) {
	code, err := r.functionRepository.GetFunction(ctx, path)
	if err != nil {
		return nil, err
	}

	var codeBuf bytes.Buffer
	if _, err := io.Copy(&codeBuf, code); err != nil {
		return nil, err
	}

	cmd := []string{"python", "-c", codeBuf.String()}

	imageName := "python:3.11-slim"

	_, err = r.dockerClient.ImageInspect(ctx, imageName)
	if err != nil {
		if errdefs.IsNotFound(err) { // ← так правильно в новых версиях
			// Образ не найден — тянем
			// r.log.Info("pulling image", zap.String("image", imageName))
			pullResp, err := r.dockerClient.ImagePull(ctx, imageName, client.ImagePullOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to pull image %s: %w", imageName, err)
			}
			io.Copy(io.Discard, pullResp)
			pullResp.Close()
		} else {
			return nil, fmt.Errorf("failed to inspect image %s: %w", imageName, err)
		}
	}

	resp, err := r.dockerClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image:        imageName,
			Cmd:          cmd,
			Tty:          false,
			AttachStdout: true,
			AttachStderr: true,
		},
		HostConfig: &container.HostConfig{
			NetworkMode: "none",
		},
	})
	if err != nil {
		return nil, err
	}

	containerID := resp.ID
	defer func() {
		r.dockerClient.ContainerRemove(context.Background(), containerID, client.ContainerRemoveOptions{})
	}()

	// 2. Start
	_, err = r.dockerClient.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	// 3. Wait с таймаутом
	ctxWait, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	wait := r.dockerClient.ContainerWait(ctxWait, containerID, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})

	select {
	case err := <-wait.Error:
		return nil, err

	case status := <-wait.Result:
		output, logErr := r.readLogs(ctx, containerID)

		if status.StatusCode != 0 {
			if logErr != nil {
				return nil, fmt.Errorf(
					"exit code %d (log read error: %w)",
					status.StatusCode,
					logErr,
				)
			}
			return output, fmt.Errorf("container exited with code %d", status.StatusCode)
		}

		return output, logErr
	}
}

func (r *Runtime) readLogs(ctx context.Context, containerID string) ([]byte, error) {
	reader, err := r.dockerClient.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	var result bytes.Buffer
	buf := make([]byte, 8)

	for {
		if _, err := io.ReadFull(reader, buf); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read log frame header: %w", err)
		}

		bodyLen := binary.BigEndian.Uint32(buf[4:8])
		if bodyLen == 0 {
			continue
		}

		body := make([]byte, bodyLen)
		if _, err := io.ReadFull(reader, body); err != nil {
			return nil, fmt.Errorf("failed to read log frame body: %w", err)
		}

		result.Write(body)
	}

	return result.Bytes(), nil
}
