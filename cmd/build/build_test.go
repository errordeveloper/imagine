package build_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
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

	h := &helper{}

	g.Expect(h.setup()).To(Succeed())

	type expectFiles map[string]struct {
		contents string
		absent   bool
	}

	for _, result := range []struct {
		args         []string
		fail         bool
		err          error
		expectRefs   map[string]string
		unexpectRefs []string
		expectFiles  expectFiles
	}{
		{
			fail: true,
			err:  fmt.Errorf(`required flag(s) "config" not set`),
		},
		{
			args: []string{"--config=non-existent"},
			fail: true,
			err:  fmt.Errorf(`unable to open config file "non-existent": %w`, &os.PathError{Op: "open", Path: "non-existent", Err: syscall.Errno(0x2)}),
		},
		{
			args: []string{
				"--config=" + testdata("sample-1.yaml"),
				"--registry=" + h.registry() + "/empty",
				"--debug",
				"--export",
				"--push",
			},
			fail: true,
			err:  fmt.Errorf("--export and --push are mutualy exclusive and cannot be set at the same time"),
		},
		{
			args: []string{
				"--config=" + testdata("sample-1.yaml"),
				"--registry=" + h.registry() + "/empty",
				"--debug",
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
			},
			unexpectRefs: []string{
				"empty/imagine-alpine-example:e4fd507.8ca1f3c",
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
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
			},
			expectRefs: map[string]string{
				"test1/imagine-alpine-example:e4fd507.8ca1f3c": "sha256:db6c90d36c7e30d2840894e035d6706a271923848c41790b82d8f40e625152a7",
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
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
			},
			expectRefs: map[string]string{
				"/test1/imagine-alpine-example:e4fd507.8ca1f3c": "sha256:db6c90d36c7e30d2840894e035d6706a271923848c41790b82d8f40e625152a7",
			},
			fail: false,
			err:  nil,
		},
		{
			args: []string{
				"--config=" + testdata("sample-2.yaml"),
				"--registry=" + h.registry() + "/test2",
				"--push",
				"--debug",
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
			},
			expectRefs: map[string]string{
				"test2/imagine-alpine-example2:a.8bdddc5.d564d0b": "",
				"test2/imagine-alpine-example2:b.8bdddc5.8ca1f3c": "",
				"test2/imagine-alpine-example2:c.8bdddc5.8ca1f3c": "",
			},
			unexpectRefs: []string{
				"test2/imagine-alpine-example2:8bdddc5.d564d0b",
				"test2/imagine-alpine-example2:8bdddc5.8ca1f3c",
			},
			fail: false,
			err:  nil,
		},
		{
			args: []string{
				"--config=" + testdata("sample-2.yaml"),
				"--registry=" + h.registry() + "/test3",
				"--export",
				"--debug",
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
			},
			unexpectRefs: []string{
				"test3/imagine-alpine-example2:a.8bdddc5.d564d0b",
				"test3/imagine-alpine-example2:b.8bdddc5.8ca1f3c",
				"test3/imagine-alpine-example2:c.8bdddc5.8ca1f3c",
				"test3/imagine-alpine-example2:8bdddc5.d564d0b",
				"test3/imagine-alpine-example2:8bdddc5.8ca1f3c",
			},
			fail: false,
			err:  nil,
			expectFiles: expectFiles{
				"image-imagine-alpine-example2-a.oci": {},
				"image-imagine-alpine-example2-b.oci": {},
				"image-imagine-alpine-example2-c.oci": {},
				"image-imagine-alpine-example2.oci":   {absent: true},
			},
		},
	} {
		cmd := BuildCmd()

		g.Expect(cmd).ToNot(BeNil())
		g.Expect(cmd.Use).To(Equal("build"))

		cmd.SetArgs(result.args)

		err := cmd.ExecuteContext(context.Background())
		if result.fail {
			g.Expect(err).To(HaveOccurred())
			g.Expect(err).To(MatchError(result.err))
		} else {
			g.Expect(err).NotTo(HaveOccurred())
			for ref, digest := range result.expectRefs {
				remoteDigest, err := crane.Digest(h.registry()+"/"+ref, crane.Insecure)
				g.Expect(err).NotTo(HaveOccurred())
				if digest != "" {
					g.Expect(remoteDigest).To(Equal(digest))
				}

			}
			for _, ref := range result.unexpectRefs {
				_, err := crane.Digest(h.registry()+"/"+ref, crane.Insecure)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("MANIFEST_UNKNOWN: manifest unknown"))
			}
			for path, file := range result.expectFiles {
				path = filepath.Join(h.repoTopLevel, path)
				if !file.absent {
					g.Expect(path).To(BeAnExistingFile())
					h.filesToRemove = append(h.filesToRemove, path)
					if file.contents != "" {
						actualOutputData, err := os.ReadFile(path)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(string(actualOutputData)).To(Equal(file.contents))
					}
				} else {
					g.Expect(path).NotTo(BeAnExistingFile())
				}
			}
		}
	}

	h.cleanup()
}

type helper struct {
	repoTopLevel, stateDir string
	buildx                 *buildx.Buildx

	filesToRemove []string

	registryHost, registryPort string
	registryContainerID        string
	docker                     *dockerClient.Client
}

func (h *helper) setup() error {
	wd, _ := os.Getwd()
	// go test runs in source dir, so we need to use
	// top-level dir due to chdir in git.New
	h.stateDir = filepath.Join(wd, "..", "..", ".imagine")
	h.repoTopLevel = filepath.Join(wd, "..", "..")

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

	buildkitdConfigPath := filepath.Join(h.stateDir, "buildkitd.toml")
	buildkitdConfigContents := fmt.Sprintf("[registry.%q]\n\thttp = true\n\tinsecure = true\n", h.registry())

	if err := os.MkdirAll(h.stateDir, 0755); err != nil {
		return err
	}
	h.filesToRemove = []string{h.stateDir}
	if err := os.WriteFile(buildkitdConfigPath, []byte(buildkitdConfigContents), 0644); err != nil {
		return err
	}

	return h.buildx.InitBuilder("", "--config="+buildkitdConfigPath)
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
	_ = h.docker.ContainerRemove(context.Background(), h.registryContainerID,
		dockerTypes.ContainerRemoveOptions{RemoveVolumes: true, Force: true})

	_ = h.buildx.Remove("")

	for _, f := range h.filesToRemove {
		_ = os.RemoveAll(f)
	}
}

func testdata(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "testdata", name)
}
