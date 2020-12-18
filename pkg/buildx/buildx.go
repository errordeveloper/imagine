package buildx

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Buildx struct {
	Builder string
	Debug   bool

	configDir      string
	managedBuilder bool
}

func New(stateDirPath string) *Buildx {
	return &Buildx{
		configDir: filepath.Join(stateDirPath, "buildx_config"),
	}
}

func (x *Buildx) Bake(filename string, args ...string) error {
	baseArgs := []string{"--file", filename, "--builder", x.Builder}

	cmd := x.mkCmd("bake", append(baseArgs, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	x.debugCmd(cmd)
	return cmd.Run()
}

func (x *Buildx) InitBuilder(existingBuilder string, platforms []string) error {
	if existingBuilder != "" {
		x.Builder = existingBuilder
		return nil
	}

	x.managedBuilder = true

	ok, err := x.UseExisting()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	x.Builder = makeBuilderName()
	return x.Create(platforms)
}

func (x *Buildx) UseExisting() (bool, error) {
	glob := filepath.Join(x.configDir, "instances", "imagine_*")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return false, err
	}
	if len(matches) == 0 {
		if x.Debug {
			fmt.Printf("zero matches for %q\n", glob)
		}
		return false, nil
	}
	if len(matches) > 1 {
		return false, fmt.Errorf("found too many matching existing builders: %v", matches)
	}

	x.Builder = filepath.Base(matches[0])

	inspectCmd := x.mkCmd("inspect", "--bootstrap", x.Builder)
	x.debugCmd(inspectCmd)
	if err := inspectCmd.Run(); err != nil {
		if exitErr, ok := errors.Unwrap(err).(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			if x.Debug {
				fmt.Printf("existing builder %q cannot be used - will remove %q and create new one\n", x.Builder, matches[0])
			}
			_ = os.RemoveAll(matches[0])

			return false, nil
		}
		return false, fmt.Errorf("failed to check if builder %q exists - %w", x.Builder, err)
	}

	useCmd := x.mkCmd("use", x.Builder)
	x.debugCmd(useCmd)
	if err := useCmd.Run(); err != nil {
		return false, fmt.Errorf("failed to use builder %q - %w", x.Builder, err)
	}
	if x.Debug {
		fmt.Printf("will use existing builder %q\n", x.Builder)
	}
	return true, nil
}

func (x *Buildx) Create(platforms []string) error {
	cmd := x.mkCmd("create", "--use", "--platform", strings.Join(platforms, ","), "--name", x.Builder)

	x.debugCmd(cmd)
	return cmd.Run()
}

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func makeBuilderName() string {
	const length = 12
	const chars = "abcdef0123456789"

	randomName := make([]byte, length)
	for i := 0; i < length; i++ {
		randomName[i] = chars[r.Intn(len(chars))]
	}
	return fmt.Sprintf("imagine_%s", string(randomName))
}
func (x *Buildx) mkCmd(cmd string, args ...string) *exec.Cmd {
	c := exec.Command("docker", append([]string{"buildx", cmd}, args...)...)
	if x.managedBuilder {
		c.Env = append(os.Environ(), "BUILDX_CONFIG="+x.configDir)
	}
	return c
}

func (x *Buildx) debugCmd(cmd *exec.Cmd) {
	if x.Debug {
		fmt.Printf("running %q\n", cmd.String())
	}
}
