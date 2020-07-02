package buildx

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func builderName() string {
	const length = 12
	const chars = "abcdef0123456789"

	randomName := make([]byte, length)
	for i := 0; i < length; i++ {
		randomName[i] = chars[r.Intn(len(chars))]
	}
	return fmt.Sprintf("imagine_%s", string(randomName))
}

type Buildx struct {
	Builder string
}

func (x *Buildx) mkCmd(cmd string, args ...string) *exec.Cmd {
	return exec.Command("docker", append([]string{"buildx", cmd}, args...)...)
}

func New() *Buildx {
	return &Buildx{
		Builder: builderName(),
	}
}

func (x *Buildx) Bake(filename string, args ...string) error {
	cmd := x.mkCmd("bake", append([]string{"--builder", x.Builder, "--file", filename}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("runing %q\n", cmd.String())
	return cmd.Run()
}

func (x *Buildx) Create() error {
	cmd := x.mkCmd("create", "--name", x.Builder)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (x *Buildx) Delete() error {
	cmd := x.mkCmd("rm", x.Builder)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
