ARG XX_IMAGE=tonistiigi/xx@sha256:23ca08d120366b31d1d7fad29283181f063b0b43879e1f93c045ca5b548868e9
ARG ALPINE_IMAGE=docker.io/library/alpine:3.17@sha256:69665d02cb32192e52e07644d76bc6f25abeb5410edc1c7a81a10ba3f0efb90a
ARG GOLANG_IMAGE=docker.io/library/golang:1.19-alpine@sha256:31c62d91ada60d52cd2e40e1134778d32549cd58c131b30dd41069f998dc8490
ARG DOCKER_IMAGE=docker.io/library/docker:23.0.0-dind@sha256:210f22afb3022a5fb994eeacd18d983c9760efd6f793ee2bf3a3467aef0c583a
ARG BUILDX_IMAGE=docker.io/docker/buildx-bin:0.10.3@sha256:62f420295e6b22901436343f743ac3d2f8b0b86917cce66c35997badff51b422

FROM --platform=$BUILDPLATFORM ${XX_IMAGE} AS xx-dist

FROM ${DOCKER_IMAGE} as docker-dist
FROM ${BUILDX_IMAGE} as buildx-dist

FROM --platform=$BUILDPLATFORM ${ALPINE_IMAGE} as rootfs

ARG TARGETARCH

COPY --from=xx-dist /usr/bin/xx-info /usr/local/bin/xx-info

RUN mkdir -p /out-${TARGETARCH}/etc/apk \
  && cp -r /etc/apk/* /out-${TARGETARCH}/etc/apk/ \
  && cp /usr/share/apk/keys/$(xx-info alpine-arch)/* /out-${TARGETARCH}/etc/apk/keys/

RUN apk add --initdb --no-cache --root /out-${TARGETARCH} --arch "$(xx-info alpine-arch)" \
      ca-certificates \
      git \
    && true


COPY <<EOF /out-${TARGETARCH}/etc/gitconfig
# this is to mitigate an issue on GitHub Actions (https://github.com/actions/checkout/issues/760)
[safe]
	directory = /github/workspace
EOF

COPY <<EOF /out-${TARGETARCH}/etc/docker/config.json
{
  "experimental": "enabled",
  "credsStore": "env"
}
EOF

FROM --platform=$BUILDPLATFORM ${GOLANG_IMAGE} as builder

ARG TARGETARCH


RUN mkdir -p /out-${TARGETARCH}/usr/local/bin

COPY --from=docker-dist /usr/local/bin /out-${TARGETARCH}/usr/local/bin
COPY --from=buildx-dist /buildx /out-${TARGETARCH}/usr/local/libexec/docker/cli-plugins/docker-buildx

WORKDIR /src/tools

RUN --mount=target=/src --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  go get github.com/errordeveloper/docker-credential-env

RUN --mount=target=/src --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} \
    go build -ldflags '-s -w' -o /out-${TARGETARCH}/usr/local/bin/docker-credential-env github.com/errordeveloper/docker-credential-env

WORKDIR /src

RUN --mount=target=/src,type=bind --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  go mod download

RUN --mount=target=/src,type=bind --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} \
    go build -ldflags '-s -w' -o /out-${TARGETARCH}/usr/local/bin/imagine ./

# TODO: ensure integration tests do run in CI

RUN --mount=target=/src,type=bind --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} \
    go test -c -ldflags '-s -w' -o /out-${TARGETARCH}/usr/local/bin/imagine.build.test ./cmd/build

RUN --mount=target=/src,type=bind --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} \
    go test -c -ldflags '-s -w' -o /out-${TARGETARCH}/usr/local/bin/imagine.generate.test ./cmd/generate

FROM --platform=$TARGETPLATFORM scratch

ARG TARGETARCH

ENV DOCKER_CONFIG=/etc/docker
COPY --from=rootfs /out-${TARGETARCH} /
COPY --from=builder /out-${TARGETARCH} /

USER 65534:65534
ENTRYPOINT [ "/usr/bin/imagine" ]
