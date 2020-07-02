package git

// inspired by: https://github.com/linuxkit/linuxkit/blob/00b9bb56a0ca46a7298964d79ce88769bef25312/src/cmd/linuxkit/pkglib/git.go

// Thin wrappers around git CLI invocations for getting
// commit and tree hashes, tags, and WIP status etc

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
)

type Git interface {
	TreeHashForHead(string) (string, error)
	CommitHashForHead(short bool) (string, error)
	TagsForHead() ([]string, error)
	SemVerTagForHead(bool) (*semver.Version, error)
	IsWIP(string) (bool, error)
	IsDev(string) (bool, error)
}

type GitRepo struct {
	repoPath string // give path of the repo, can be relative
	TopLevel string // actual path of the repo as seen by git
}

func NewFromCWD() (*GitRepo, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return New(wd)
}

func New(repoPath string) (*GitRepo, error) {
	g := &GitRepo{repoPath: repoPath}

	ok, err := g.isWorkTree()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("directory %s is not in git", repoPath)
	}

	ok, err = g.isAtTopLevel()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("current working directory is not at repo top level")
	}

	return g, nil
}

func (g *GitRepo) mkCmd(args ...string) *exec.Cmd {
	return exec.Command("git", append([]string{"-C", g.repoPath}, args...)...)
}

func (g *GitRepo) commandStdout(stderr io.Writer, args ...string) (string, error) {
	cmd := g.mkCmd(args...)
	cmd.Stderr = stderr

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (g *GitRepo) command(args ...string) error {
	cmd := g.mkCmd(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (g *GitRepo) isWorkTree() (bool, error) {
	revParseOut, err := g.commandStdout(nil, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	result := strings.TrimSpace(revParseOut)

	if result == "true" {
		return true, nil
	}

	return false, fmt.Errorf("unexpected output from git rev-parse --is-inside-work-tree: %s", result)
}

func (g *GitRepo) isAtTopLevel() (bool, error) {
	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	revParseOut, err := g.commandStdout(nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return false, err
	}
	g.TopLevel = strings.TrimSpace(revParseOut)
	return g.TopLevel == wd, nil
}

func (g *GitRepo) TreeHashForHead(path string) (string, error) {
	revParseOut, err := g.commandStdout(os.Stderr, "rev-parse", "HEAD:"+path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(revParseOut), nil
}

func (g *GitRepo) CommitHashForHead(short bool) (string, error) {
	args := []string{"rev-parse", "HEAD"}
	if short {
		args = []string{"rev-parse", "--short", "HEAD"}
	}
	out, err := g.commandStdout(os.Stderr, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

func (g *GitRepo) TagsForHead() ([]string, error) {
	// using name-rev provides clear indication in case there is no tag
	nameRevOut, err := g.commandStdout(os.Stderr, "name-rev", "--name-only", "--no-undefined", "--tags", "HEAD")
	if err != nil {
		return nil, err
	}

	// name-rev returns results in `^0` notation, so to get actual tag
	// back, we call tag command
	tagOut, err := g.commandStdout(os.Stderr, "tag", "--sort", "tag", "--points-at", strings.TrimSpace(nameRevOut))
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(tagOut), "\n"), nil
}

func semVerFromTags(ignoreParserErrors bool, tags []string) (*semver.Version, error) {
	versions := []*semver.Version{}
	for _, t := range tags {
		version, err := semver.NewVersion(t)
		if err != nil {
			if ignoreParserErrors {
				continue
			}
			return nil, err
		}
		versions = append(versions, version)
	}
	l := len(versions)
	if l == 0 {
		return nil, fmt.Errorf("no version tags found")
	}
	if l == 1 {
		return versions[0], nil
	}

	// in case of multiple semver tags are pointed to the same
	// commit, return highest semver
	sort.Sort(semver.Collection(versions))
	return versions[l-1], nil
}

func (g *GitRepo) SemVerTagForHead(ignoreParserErrors bool) (*semver.Version, error) {
	tags, err := g.TagsForHead()
	if err != nil {
		return nil, err
	}

	return semVerFromTags(ignoreParserErrors, tags)
}

func (g *GitRepo) IsWIP(path string) (bool, error) {
	// update cache, otherwise files which have an updated
	// timestamp but no actual changes are marked as changes
	// because `git diff-index` only uses the `lstat` result and
	// not the actual file contents
	if err := g.command("update-index", "-q", "--refresh"); err != nil {
		return false, err
	}

	err := g.command("diff-index", "--quiet", "HEAD", "--", path)
	if err == nil {
		return false, nil
	}
	switch err.(type) {
	case *exec.ExitError:
		// diff-index exits with an error if there are differences
		return true, nil
	default:
		return false, err
	}
}

func (g *GitRepo) IsDev(baseBranch string) (bool, error) {
	revParseOut, err := g.commandStdout(os.Stderr, "rev-parse", "HEAD")
	if err != nil {
		return false, err
	}

	_, err = g.commandStdout(os.Stderr, "merge-base", "--is-ancestor", strings.TrimSpace(revParseOut), baseBranch)
	if err != nil {
		return false, err
	}

	return true, nil
}

type FakeRepo struct {
	TreeHashForHeadRoot  string
	TreeHashForHeadVal   map[string]string
	CommitHashForHeadVal string
	TagsForHeadVal       []string
	IsWIPVal             map[string]bool
	IsWIPRoot            bool
	IsDevVal             bool
}

func (f *FakeRepo) TreeHashForHead(path string) (string, error) {
	if path == "" {
		return f.TreeHashForHeadRoot, nil
	}
	v, ok := f.TreeHashForHeadVal[path]
	if !ok {
		return "", fmt.Errorf("%s not in fake tree", path)
	}
	return v, nil
}

func (f *FakeRepo) CommitHashForHead(short bool) (string, error) {
	if short {
		return f.CommitHashForHeadVal[:6], nil
	}
	return f.CommitHashForHeadVal, nil
}

func (f *FakeRepo) TagsForHead() ([]string, error) {
	if len(f.TagsForHeadVal) == 0 {
		return nil, fmt.Errorf("no tag in fake repo")
	}
	return f.TagsForHeadVal, nil
}

func (f *FakeRepo) SemVerTagForHead(ignoreParserErrors bool) (*semver.Version, error) {
	tags, err := f.TagsForHead()
	if err != nil {
		return nil, err
	}
	return semVerFromTags(ignoreParserErrors, tags)
}

func (f *FakeRepo) IsWIP(path string) (bool, error) {
	if path == "" {
		return f.IsWIPRoot, nil
	}
	v, ok := f.IsWIPVal[path]
	if !ok {
		return false, fmt.Errorf("%s not in fake tree", path)
	}
	return v, nil
}

func (f *FakeRepo) IsDev(string) (bool, error) {
	return f.IsDevVal, nil
}
