package git

import (
	"fmt"

	"github.com/Masterminds/semver"
)

type FakeRepo struct {
	TreeHashForHeadRoot  string
	TreeHashForHeadVal   map[string]string
	CommitHashForHeadVal string
	TagsForHeadVal       []string
	IsWIPVal             map[string]bool
	IsWIPRoot            bool
	IsDevVal             bool
}

func getHash(hash string, short bool) (string, error) {
	if short {
		return hash[:6], nil
	}
	return hash, nil
}

func (f *FakeRepo) TreeHashForHead(path string, short bool) (string, error) {
	if path == "" {
		return getHash(f.TreeHashForHeadRoot, short)
	}
	v, ok := f.TreeHashForHeadVal[path]
	if !ok {
		return "", fmt.Errorf("%s not in fake tree (TreeHashForHead)", path)
	}
	return getHash(v, short)
}

func (f *FakeRepo) CommitHashForHead(short bool) (string, error) {
	return getHash(f.CommitHashForHeadVal, short)
}

func (f *FakeRepo) TagsForHead() ([]string, error) {
	if len(f.TagsForHeadVal) == 0 {
		return nil, fmt.Errorf("no tag in fake repo (TagsForHead)")
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
		return false, fmt.Errorf("%s not in fake tree (IsWIP)", path)
	}
	return v, nil
}

func (f *FakeRepo) IsDev(string) (bool, error) {
	return f.IsDevVal, nil
}
