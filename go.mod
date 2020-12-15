module github.com/errordeveloper/imagine

require (
	github.com/Masterminds/semver v1.5.0
	github.com/docker/buildx v0.5.1
	github.com/google/go-containerregistry v0.1.2
	github.com/onsi/gomega v1.9.0
	github.com/spf13/cobra v1.0.0
)

// based on https://github.com/docker/buildx/blob/v0.5.1/go.mod#L61-L68

replace (
	// protobuf: corresponds to containerd (through buildkit)
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.5
	github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305

	// genproto: corresponds to containerd (through buildkit)
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200224152610-e50cd9704f63
)

go 1.14
