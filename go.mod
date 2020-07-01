module github.com/errordeveloper/imagine

require (
	github.com/docker/buildx v0.4.1
	github.com/onsi/gomega v1.7.1
)

// based on https://github.com/docker/buildx/blob/f3111bcbef8ce7e3933711358419fa18294b3daf/go.mod#L69-L73

replace github.com/containerd/containerd => github.com/containerd/containerd v1.3.1-0.20200227195959-4d242818bf55

replace github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200227233006-38f52c9fec82

replace github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305

go 1.14
