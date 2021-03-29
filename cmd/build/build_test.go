package build_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/errordeveloper/imagine/cmd/build"
	"github.com/errordeveloper/imagine/pkg/buildx"

	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

func TestBuildCmd(t *testing.T) {
	g := NewGomegaWithT(t)

	//g.Expect(os.Setenv(buildx.EnvImagineBuildxCommamnd, "./dummy-buildx.sh")).To(Succeed())

	h := &helper{}

	g.Expect(h.initBuilder()).To(Succeed())

	cmd := BuildCmd()

	g.Expect(cmd).ToNot(BeNil())
	g.Expect(cmd.Use).To(Equal("build"))

	for _, result := range []struct {
		args []string
		fail bool
		err  error
	}{
		{
			fail: true,
			err:  fmt.Errorf(`required flag(s) "config" not set`),
		},
		{
			args: []string{"--config=non-existent"},
			fail: true,
			err:  &os.PathError{Op: "open", Path: "non-existent", Err: syscall.Errno(0x2)},
		},
		{
			args: []string{
				"--config=" + testdata("sample-1.yaml"),
				"--registry=" + h.registry() + "/empty",
				"--debug",
			},
			fail: false,
			err:  nil,
		},
		{
			args: []string{
				"--config=" + testdata("sample-1.yaml"),
				"--registry=" + h.registry() + "/test1",
				"--push",
				"--debug",
			},
			fail: false,
			err:  nil,
		},
		{
			args: []string{
				"--config=" + testdata("sample-1.yaml"),
				"--registry=" + h.registry() + "/test1",
				"--push",
				"--debug",
			},
			fail: false,
			err:  nil,
		},
	} {
		cmd.SetArgs(result.args)
		err := cmd.ExecuteContext(context.Background())
		if result.fail {
			g.Expect(err).To(HaveOccurred())
			g.Expect(err).To(MatchError(result.err))
		} else {
			g.Expect(err).ToNot(HaveOccurred())
		}
		// TODO: check registry
	}

	h.cleanup()
}

type helper struct {
	stateDir, registryHost, registryPort, registryContainerID, buildkitdConfigPath string

	docker *dockerClient.Client
	buildx *buildx.Buildx
}

func (h *helper) initBuilder() error {
	wd, _ := os.Getwd()
	// go test runs in source dir, so we need to use
	// top-level dir due to chdir in git.New
	h.stateDir = filepath.Join(wd, "..", "..", ".imagine")

	h.buildx = buildx.New(h.stateDir)
	h.buildx.Debug = true

	builderDescPath, err := h.buildx.FindExisting()
	if err != nil {
		return err
	}
	if builderDescPath != "" {
		_ = h.buildx.Remove(builderDescPath)
		_ = os.RemoveAll(h.stateDir)
	}

	if err := h.startRegistry(); err != nil {
		return err
	}

	h.buildkitdConfigPath = filepath.Join(h.stateDir, "buildkitd.toml")
	buildkitdConfigContents := fmt.Sprintf("[registry.%q]\nhttp = true\ninsecure = true", h.registry())

	if err := os.WriteFile(h.buildkitdConfigPath, []byte(buildkitdConfigContents), 0644); err != nil {
		return err
	}

	return h.buildx.InitBuilder("", "--config="+h.buildkitdConfigPath)
}

func (h *helper) startRegistry() error {
	ctx := context.Background()

	docker, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	config := &dockerContainer.Config{
		Image:        "registry:2@sha256:9a2d4585a32de2df9aadc126708edd1da5f875093db6419d6894ddc2d1115d97",
		ExposedPorts: nat.PortSet{"5000/tcp": struct{}{}},
	}

	if _, err := docker.ImagePull(ctx, config.Image, dockerTypes.ImagePullOptions{}); err != nil {
		return err
	}

	resp, err := docker.ContainerCreate(ctx, config, &dockerContainer.HostConfig{PublishAllPorts: true}, nil, nil, "")
	if err != nil {
		return err
	}

	if err := docker.ContainerStart(ctx, resp.ID, dockerTypes.ContainerStartOptions{}); err != nil {
		return err
	}

	h.registryContainerID = resp.ID

	containerInfo, err := docker.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return err
	}

	h.registryHost = "host.docker.internal" // TODO: this works on Docker for Desktop, check if it works in GitHub Actions
	h.registryPort = containerInfo.NetworkSettings.Ports["5000/tcp"][0].HostPort

	h.docker = docker

	return nil
}
func (h *helper) registry() string {
	return fmt.Sprintf("%s:%s", h.registryHost, h.registryPort)
}

func (h *helper) cleanup() {
	_ = os.Remove(h.buildkitdConfigPath)

	_ = h.docker.ContainerRemove(context.Background(), h.registryContainerID,
		dockerTypes.ContainerRemoveOptions{RemoveVolumes: true, Force: true})

	_ = h.buildx.Remove("")

	_ = os.RemoveAll(h.stateDir)
}

func testdata(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "testdata", name)
}
