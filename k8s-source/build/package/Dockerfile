# Build the source binary
FROM golang:1.24-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_VERSION
ARG BUILD_COMMIT

# required for accessing the private dependencies and generating version descriptor
RUN apk add --no-cache git curl

WORKDIR /workspace

# Copy the go source
COPY . .

# Build
RUN --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w -X github.com/overmindtech/cli/tracing.version=${BUILD_VERSION} -X github.com/overmindtech/cli/tracing.commit=${BUILD_COMMIT}" -o source k8s-source/main.go

FROM alpine:3.21
WORKDIR /
COPY --from=builder /workspace/source .
USER 65534:65534

ENTRYPOINT ["/source"]
