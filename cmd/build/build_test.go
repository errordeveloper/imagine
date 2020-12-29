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
)

func TestBuildCmd(t *testing.T) {
	g := NewGomegaWithT(t)

	//g.Expect(os.Setenv(buildx.EnvImagineBuildxCommamnd, "./dummy-buildx.sh")).To(Succeed())

	g.Expect(initBuilder()).To(Succeed())

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
				"--registry=host.docker.internal:5000/empty",
				"--debug",
			},
			fail: false,
			err:  nil,
		},
		{
			args: []string{
				"--config=" + testdata("sample-1.yaml"),
				"--registry=host.docker.internal:5000/test1",
				"--push",
				"--debug",
			},
			fail: false,
			err:  nil,
		},
		{
			args: []string{
				"--config=" + testdata("sample-1.yaml"),
				"--registry=host.docker.internal:5000/test1",
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
}

func initBuilder() error {
	wd, _ := os.Getwd()
	// go test runs in source dir, so we need to use
	// top-level dir due to chdir in git.New
	stateDir := filepath.Join(wd, "..", "..", ".imagine")

	bx := buildx.New(stateDir)
	bx.Debug = true

	builderDescPath, err := bx.FindExisting()
	if err != nil {
		return err
	}
	if builderDescPath != "" {
		_ = bx.Remove(builderDescPath)
		_ = os.RemoveAll(stateDir)
	}

	return bx.InitBuilder("", "--config="+testdata("buildkitd.toml"))
}

func testdata(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "testdata", name)
}
