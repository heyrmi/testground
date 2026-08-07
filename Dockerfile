# syntax=docker/dockerfile:1

# The frontend bundle is committed and compiled into the binary by go:embed,
# exactly as it is for `go install`, so nothing here needs a Node toolchain.
# The consequence is worth stating plainly: this copies web/app/dist, it does
# not rebuild it, so an image built while iterating on web/app serves the last
# bundle someone ran `make web` over.
FROM golang:1.23-alpine AS build

WORKDIR /src

# Modules before source. The dependency graph changes far less often than the
# challenge files do, and without the split every edit to a challenge would
# re-download the whole graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# These flags mirror .goreleaser.yaml deliberately. An image built from source
# and an image built from a release artefact have to be linked the same way,
# or a fault that reproduces in only one of them has no explanation anyone can
# find.
RUN CGO_ENABLED=0 go build -trimpath \
	-ldflags="-s -w \
	-X github.com/heyrmi/testground/internal/build.Version=${VERSION} \
	-X github.com/heyrmi/testground/internal/build.Commit=${COMMIT} \
	-X github.com/heyrmi/testground/internal/build.Date=${DATE}" \
	-o /out/playground ./cmd/playground

# Templates, the vendored stylesheets, the frontend bundle and the fixture
# media all live inside the binary, and the playground makes no outbound
# request at all -- offline operation is a promise the project makes. So the
# runtime needs no shell, no libc and no package manager. Distroless static is
# chosen over scratch for one reason: it ships /etc/passwd with a nonroot
# entry, which lets the user below be a name rather than a bare uid that
# `docker top` and cluster policy both have to guess at.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

LABEL org.opencontainers.image.title="testground" \
	org.opencontainers.image.description="A self-contained playground of browser-automation challenges for QA engineers: pages that break naive locators, waits and assertions on purpose." \
	org.opencontainers.image.source="https://github.com/heyrmi/testground" \
	org.opencontainers.image.url="https://github.com/heyrmi/testground" \
	org.opencontainers.image.documentation="https://github.com/heyrmi/testground/blob/main/README.md" \
	org.opencontainers.image.licenses="MIT" \
	org.opencontainers.image.version="${VERSION}" \
	org.opencontainers.image.revision="${COMMIT}" \
	org.opencontainers.image.created="${DATE}"

# Both ports, because the browser decides what is same-origin from the port and
# the frame challenges embed a genuinely different one. Publish only 7373 and
# those pages still render, with an iframe pointing at a socket nothing
# answers -- a broken challenge rather than an absent one.
EXPOSE 7373 7374

USER nonroot:nonroot

# There is deliberately no HEALTHCHECK. Probing /api/health from inside would
# mean adding a shell or a curl, and carrying a package manager in the image so
# that the image can poll itself is a worse trade than letting whatever runs
# the container probe the port from outside.
ENTRYPOINT ["/usr/local/bin/playground"]

# The binary binds loopback by default, which is the right default on a laptop
# and useless in a container: loopback inside the network namespace is not what
# a published port forwards to, so the mapping would succeed and every request
# would be refused. Overriding it in CMD rather than changing the flag's
# default keeps the safe default for everyone outside a container, and leaves
# this replaceable by anything `docker run` is given after the image name.
CMD ["serve", "--addr", "0.0.0.0:7373", "--cross-origin-addr", "0.0.0.0:7374"]

# GoReleaser has already cross-compiled the binary by the time it builds an
# image, and the context it hands to Docker holds that binary and nothing else.
# There is no source tree to compile here, which is why this stage is reached
# only through an explicit --target and is not the last one in the file.
FROM runtime AS release

COPY playground /usr/local/bin/playground

# Last, so that it is the default target and `docker build .` at the repo root
# works with no flags and nothing but this checkout.
FROM runtime AS standalone

COPY --from=build /out/playground /usr/local/bin/playground
