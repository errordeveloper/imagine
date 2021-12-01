module github.com/errordeveloper/imagine

require (
	github.com/Masterminds/semver v1.5.0
	github.com/docker/buildx v0.7.1
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/google/go-containerregistry v0.1.2
	github.com/onsi/gomega v1.10.3
	github.com/spf13/cobra v1.2.1
	sigs.k8s.io/yaml v1.2.0
)

// based on https://github.com/docker/buildx/blob/v0.7.1/go.mod#L58-L65

replace (
	github.com/docker/cli => github.com/docker/cli v20.10.3-0.20210702143511-f782d1355eff+incompatible
	github.com/docker/docker => github.com/tonistiigi/docker v0.10.1-0.20211122204227-65a6f25dbca2
	github.com/tonistiigi/fsutil => github.com/tonistiigi/fsutil v0.0.0-20211122210416-da5201e0b3af
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => github.com/tonistiigi/opentelemetry-go-contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.0.0-20210714055410-d010b05b4939
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace => github.com/tonistiigi/opentelemetry-go-contrib/instrumentation/net/http/httptrace/otelhttptrace v0.0.0-20210714055410-d010b05b4939
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => github.com/tonistiigi/opentelemetry-go-contrib/instrumentation/net/http/otelhttp v0.0.0-20210714055410-d010b05b4939
)

go 1.14
