module github.com/errordeveloper/imagine

require (
	github.com/Masterminds/semver v1.5.0
	github.com/docker/buildx v0.6.1
	github.com/google/go-containerregistry v0.1.2
	github.com/onsi/gomega v1.9.0
	github.com/spf13/cobra v1.0.0
)

// based on https://github.com/docker/buildx/blob/v0.6.1/go.mod#L64-L65

replace (
	github.com/docker/cli => github.com/docker/cli v20.10.3-0.20210702143511-f782d1355eff+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20210609100121-ef4d47340142+incompatible
)

go 1.14
