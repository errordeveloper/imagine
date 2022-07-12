package generate_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/errordeveloper/imagine/cmd/generate"
	"github.com/errordeveloper/imagine/pkg/recipe"
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
			err:  fmt.Errorf(`unable to open config file "non-existent": %w`, &os.PathError{Op: "open", Path: "non-existent", Err: syscall.Errno(0x2)}),
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
				"--export",
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
		}, {
			args: []string{
				"--config=" + testdata("sample-4-bad.yaml"),
			},
			fail: true,
			err: fmt.Errorf(`config file "cmd/testdata/sample-4-bad.yaml" is invalid: %w`,
				fmt.Errorf(`at least '.spec.dir' or '.spec.variants' must be set`)),
		}, {
			args: []string{
				"--config=" + testdata("sample-5-bad.yaml"),
			},
			fail: true,
			err: fmt.Errorf(`config file "cmd/testdata/sample-5-bad.yaml" is invalid: %w`,
				fmt.Errorf(`'.spec.name' must be set`)),
		},
		// TODO: cover some error-cases for files not in git git
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
				matchFixture(g, result.actualOutput, result.expectedOutput)
			}
		}
	}

	h.cleanup()
}

func matchFixture(g *WithT, actualOutput, expectedOutput string) {
	actualOutputData, err := os.ReadFile(actualOutput)
	g.Expect(err).NotTo(HaveOccurred())
	actualOutputObj := &recipe.BakeManifest{}
	g.Expect(json.Unmarshal(actualOutputData, actualOutputObj)).To(Succeed())

	expectedOutputData, err := os.ReadFile(expectedOutput)
	g.Expect(err).NotTo(HaveOccurred())
	expectedOutputObj := &recipe.BakeManifest{}
	g.Expect(json.Unmarshal(expectedOutputData, expectedOutputObj)).To(Succeed())

	for k := range expectedOutputObj.Target {
		if strings.HasPrefix(k, recipe.IndexTargetNamePrefix) {
			// index tag changes all the time, so it needs to be igrnored
			expectedOutputObj.Target[k].Tags = actualOutputObj.Target[k].Tags
		}
		replaceCheckoutPrefix(expectedOutputObj.Target[k].Dockerfile)
		replaceCheckoutPrefix(expectedOutputObj.Target[k].Context)
		for i := range expectedOutputObj.Target[k].Outputs {
			replaceCheckoutPrefix(&expectedOutputObj.Target[k].Outputs[i])
		}
	}

	actualOutputData, err = json.Marshal(actualOutputObj)
	g.Expect(err).NotTo(HaveOccurred())
	expectedOutputData, err = json.Marshal(expectedOutputObj)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(actualOutputData).To(MatchJSON(expectedOutputData), "fixture: "+expectedOutput)
}

func replaceCheckoutPrefix(v *string) {
	// due to imagine calling chdir, workdir is already repo root
	wd, _ := os.Getwd()
	if v != nil {
		*v = strings.Replace(*v, "${CHECKOUT_PREFIX}", wd, 1)
	}
}

type helper struct {
	stateDir string
}

func (h *helper) setup() error {
	wd, _ := os.Getwd()
	// go test runs in source dir, so we need to use
	// top-level dir due to chdir in git.New, it's only later
	// that imagine does chdir to repo root
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
