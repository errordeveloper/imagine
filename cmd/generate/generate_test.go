package generate_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/errordeveloper/imagine/cmd/generate"
)

func TestGenerateCmd(t *testing.T) {
	g := NewGomegaWithT(t)

	h := &helper{}

	g.Expect(h.setup()).To(Succeed())

	for _, result := range []struct {
		args []string
		fail bool
		err  error

		expectedOutput, actualOutput string
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
				"--registry=example.com/test1",
				"--debug",
				"--output=" + h.stateDir + "/generate-0.json",
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
			},
			fail: false,
			err:  nil,

			expectedOutput: testdata("generate-0-imagine-alpine-example.json"),
			actualOutput:   h.stateDir + "/generate-0.json",
		},
		{
			args: []string{
				"--config=" + testdata("sample-2.yaml"),
				"--registry=example.com/test2",
				"--push", "--export", // TODO: make it fail when --export is set, add tests,
				"--debug",
				"--platform=plan9/sparc",
				"--platform=netbsd/toaster",
				"--output=" + h.stateDir + "/generate-1.json",
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
			},
			fail: false,
			err:  nil,

			expectedOutput: testdata("generate-1-imagine-alpine-example2.json"),
			actualOutput:   h.stateDir + "/generate-1.json",
		},
		{
			args: []string{
				"--config=" + testdata("sample-3.yaml"),
				"--registry=example.com/test3",
				"--push",
				"--debug",
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
				"--output=" + h.stateDir + "/generate-2.json",
			},
			fail: false,
			err:  nil,

			expectedOutput: testdata("generate-2-imagine-alpine-example2.json"),
			actualOutput:   h.stateDir + "/generate-2.json",
		},
		{
			args: []string{
				"--config=" + testdata("sample-3.yaml"),
				"--registry=example.com/test3",
				"--debug",
				"--export",
				// TODO: avoid having to set this (it is to make sure tests pass on any branch)
				"--without-tag-suffix",
				"--output=" + h.stateDir + "/generate-3.json",
			},
			fail: false,
			err:  nil,

			expectedOutput: testdata("generate-3-imagine-alpine-example2.json"),
			actualOutput:   h.stateDir + "/generate-3.json",
		},
	} {
		cmd := GenerateCmd()

		g.Expect(cmd).ToNot(BeNil())
		g.Expect(cmd.Use).To(Equal("generate"))

		cmd.SetArgs(result.args)

		err := cmd.ExecuteContext(context.Background())
		if result.fail {
			g.Expect(err).To(HaveOccurred())
			g.Expect(err).To(MatchError(result.err))
		} else {
			g.Expect(err).NotTo(HaveOccurred())
			if result.expectedOutput != "" {
				expectedOutputData, err := os.ReadFile(result.expectedOutput)
				g.Expect(err).NotTo(HaveOccurred())
				actualOutputData, err := os.ReadFile(result.actualOutput)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(actualOutputData).To(MatchJSON(expectedOutputData))
			}
		}
	}

	h.cleanup()
}

type helper struct {
	stateDir string
}

func (h *helper) setup() error {
	wd, _ := os.Getwd()
	// go test runs in source dir, so we need to use
	// top-level dir due to chdir in git.New
	h.stateDir = filepath.Join(wd, "..", "..", ".imagine")

	if err := os.MkdirAll(h.stateDir, 0755); err != nil {
		return err
	}

	return nil
}

func (h *helper) cleanup() {
	_ = os.RemoveAll(h.stateDir)
}

func testdata(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "testdata", name)
}
