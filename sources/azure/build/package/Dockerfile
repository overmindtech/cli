# Build the source binary
FROM golang:1.26-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_VERSION
ARG BUILD_COMMIT

# required for generating the version descriptor
RUN apk upgrade --no-cache && apk add --no-cache git

WORKDIR /workspace

# Copy the go source
COPY . .

# Build
RUN --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w -X github.com/overmindtech/cli/go/tracing.version=${BUILD_VERSION} -X github.com/overmindtech/cli/go/tracing.commit=${BUILD_COMMIT}" -o source sources/azure/main.go

FROM alpine:3.23.3
WORKDIR /
COPY --from=builder /workspace/source .
USER 65534:65534

ENTRYPOINT ["/source"]
