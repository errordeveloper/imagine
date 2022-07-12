ARG XX_IMAGE=tonistiigi/xx@sha256:23ca08d120366b31d1d7fad29283181f063b0b43879e1f93c045ca5b548868e9
ARG ALPINE_IMAGE=docker.io/library/alpine:3.16@sha256:686d8c9dfa6f3ccfc8230bc3178d23f84eeaf7e457f36f271ab1acc53015037c
ARG GOLANG_IMAGE=docker.io/library/golang:1.18.3-alpine@sha256:7cc62574fcf9c5fb87ad42a9789d5539a6a085971d58ee75dd2ee146cb8a8695
ARG DOCKER_IMAGE=docker.io/library/docker:20.10.17-dind@sha256:6dd895238f055a79a6d044f7d97b668bef0f9a840e5eed06fa01f1a6b7aed17e
ARG BUILDX_IMAGE=docker.io/docker/buildx-bin:0.8.2@sha256:44a7217da633dd01f621ebcfe75410cd98f81edcff5b43de33eeb65582d6f055


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
    && true

FROM --platform=$BUILDPLATFORM ${GOLANG_IMAGE} as builder

ARG TARGETARCH


RUN mkdir -p /out-${TARGETARCH}/usr/local/bin

COPY --from=docker-dist /usr/local/bin /out-${TARGETARCH}/usr/local/bin
COPY --from=buildx-dist /buildx /out-${TARGETARCH}/usr/local/libexec/docker/cli-plugins/docker-buildx

WORKDIR /tmp

RUN --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} GOPATH=/out-${TARGETARCH}/usr/local/bin \
    go install github.com/errordeveloper/docker-credential-env@v0.1.5

WORKDIR /src

RUN --mount=target=/src,type=bind --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  go mod download

RUN --mount=target=/src,type=bind --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} \
    go build -ldflags '-s -w' -o /out-${TARGETARCH}/usr/local/bin/imagine ./

FROM --platform=$TARGETPLATFORM scratch

ARG TARGETARCH

COPY --from=rootfs /out-${TARGETARCH} /
COPY --from=builder /out-${TARGETARCH} /

ENTRYPOINT [ "/usr/bin/imagine" ]
