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

	Platforms []string

	configDir      string
	managedBuilder bool
}

const (
	EnvImagineBuildxCommamnd = "IMAGINE_BUILDX_COMMAND"
	EnvBuildxConfig          = "BUILDX_CONFIG"
	buildxConfigDirName      = "buildx_config"
)

func New(stateDirPath string) *Buildx {
	return &Buildx{
		configDir: filepath.Join(stateDirPath, buildxConfigDirName),
	}
}

func (x *Buildx) Bake(filename string, args ...string) error {
	cmd := x.mkCmd("bake", "--file", filename, "--builder", x.Builder)
	cmd.Args = append(cmd.Args, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	x.debugCmd(cmd)
	return cmd.Run()
}

func (x *Buildx) InitBuilder(existingBuilder string, args ...string) error {
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
	return x.Create(args...)
}

func (x *Buildx) FindExisting() (string, error) {
	glob := filepath.Join(x.configDir, "instances", "imagine_*")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		if x.Debug {
			fmt.Printf("zero matches for %q\n", glob)
		}
		return "", nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("found too many matching existing builders: %v", matches)
	}
	return matches[0], nil
}

func (x *Buildx) UseExisting() (bool, error) {
	builderDescPath, err := x.FindExisting()

	if err != nil {
		return false, err
	}

	if builderDescPath == "" {
		return false, nil
	}

	x.Builder = filepath.Base(builderDescPath)

	inspectCmd := x.mkCmd("inspect", "--bootstrap", x.Builder)
	x.debugCmd(inspectCmd)
	if inspectErr := inspectCmd.Run(); inspectErr != nil {
		if inspectExitErr, ok := errors.Unwrap(inspectErr).(*exec.ExitError); ok && inspectExitErr.ExitCode() == 1 {
			if x.Debug {
				fmt.Printf("existing builder %q cannot be used - will remove %q and create new one\n", x.Builder, builderDescPath)
			}

			if err := x.Remove(builderDescPath); err != nil && x.Debug {
				fmt.Printf("error while cleaning up buildkit instace: %s\n", err.Error())
			}
			return false, nil
		}
		return false, fmt.Errorf("failed to check if builder %q exists - %w", x.Builder, inspectErr)
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

func (x *Buildx) Create(args ...string) error {
	cmd := x.mkCmd("create", "--use", "--name", x.Builder)
	if len(x.Platforms) > 0 {
		cmd.Args = append(cmd.Args, "--platform", strings.Join(x.Platforms, ","))
	}
	cmd.Args = append(cmd.Args, args...)
	x.debugCmd(cmd)
	return cmd.Run()
}

func (x *Buildx) Remove(builderDescPath string) error {
	builder := x.Builder
	if builderDescPath != "" {
		builder = filepath.Base(builderDescPath)
	}
	if builder == "" {
		return fmt.Errorf("neither x.Builder or builderDescPath was specified")
	}
	cmd := x.mkCmd("rm", builder)
	x.debugCmd(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}
	if builderDescPath != "" {
		return os.RemoveAll(builderDescPath)
	}
	return nil
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

func (x *Buildx) getBaseCmd(cmd string) *exec.Cmd {
	if overrideBuildxCmd := os.Getenv(EnvImagineBuildxCommamnd); overrideBuildxCmd != "" {
		if x.Debug {
			fmt.Printf("will use %q\n", overrideBuildxCmd)
		}
		return exec.Command(overrideBuildxCmd, cmd)
	}
	return exec.Command("docker", "buildx", cmd)
}

func (x *Buildx) mkCmd(cmd string, args ...string) *exec.Cmd {
	c := x.getBaseCmd(cmd)
	c.Args = append(c.Args, args...)
	if x.managedBuilder {
		c.Env = append(os.Environ(), EnvBuildxConfig+"="+x.configDir)
	}
	return c
}

func (x *Buildx) debugCmd(cmd *exec.Cmd) {
	if x.Debug {
		fmt.Printf("running %q\n", cmd.String())
	}
}
