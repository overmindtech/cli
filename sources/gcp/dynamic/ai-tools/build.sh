#!/bin/bash

# Build script for prompter tools
# Usage: ./build.sh [platform] [tool]
# Examples:
#   ./build.sh                    # Build both tools for current platform
#   ./build.sh linux/amd64        # Build both tools for Linux AMD64
#   ./build.sh darwin/arm64       # Build both tools for macOS ARM64
#   ./build.sh "" generate-test-ticket  # Build only generate-test-ticket for current platform
#   ./build.sh linux/amd64 generate-adapter-ticket     # Build only generate-adapter-ticket for Linux AMD64

set -e

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

PLATFORM="${1:-}"
TOOL="${2:-}"

# Define available tools
TOOLS=("generate-test-ticket" "generate-adapter-ticket")

# If specific tool requested, validate it
if [ -n "$TOOL" ]; then
    if [[ ! " ${TOOLS[@]} " =~ " ${TOOL} " ]]; then
        echo "Error: Unknown tool '$TOOL'. Available tools: ${TOOLS[*]}"
        exit 1
    fi
    TOOLS=("$TOOL")
fi

# Build function
build_tool() {
    local tool="$1"
    local platform="$2"
    local source_dir="${tool}-cmd"
    local binary_name="$tool"

    if [ ! -d "$source_dir" ]; then
        echo "Error: Source directory '$source_dir' not found"
        return 1
    fi

    if [ -z "$platform" ]; then
        echo "Building $binary_name for current platform..."
        go build -o "$binary_name" "./$source_dir"
        echo "✅ Built successfully: $binary_name"
    else
        echo "Building $binary_name for $platform..."

        # Split platform into GOOS and GOARCH
        IFS='/' read -r GOOS GOARCH <<< "$platform"

        if [ -z "$GOOS" ] || [ -z "$GOARCH" ]; then
            echo "Error: Invalid platform format. Use: os/arch (e.g., linux/amd64)"
            return 1
        fi

        OUTPUT_NAME="${binary_name}-${GOOS}-${GOARCH}"
        if [ "$GOOS" = "windows" ]; then
            OUTPUT_NAME="${OUTPUT_NAME}.exe"
        fi

        GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$OUTPUT_NAME" "./$source_dir"
        echo "✅ Built successfully: $OUTPUT_NAME"
    fi
}

# Build all requested tools
for tool in "${TOOLS[@]}"; do
    build_tool "$tool" "$PLATFORM"
done

echo ""
echo "Built tools:"
for tool in "${TOOLS[@]}"; do
    echo "  $tool"
done

echo ""
echo "Usage examples:"
echo "  ./generate-test-ticket [--verbose|-v] <adapter-name>"
echo "  ./generate-adapter-ticket -name monitoring-alert-policy -api-ref https://..."
echo ""
echo "For more information, see README.md"
