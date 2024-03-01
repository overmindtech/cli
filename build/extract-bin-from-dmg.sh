#!/bin/bash

# Ensure BUILD_NAME is provided as the first argument
if [ $# -lt 1 ]; then
  echo "Usage: $0 <build-name>"
  exit 1
fi

BUILD_NAME="$1" # Set BUILD_NAME from the script argument
BUILD_PATH="dist/signed-binaries"

# Define source DMG file and destination directory
SOURCE_DMG="dist/${BUILD_NAME}.dmg"

echo "Starting the process with BUILD_NAME: $BUILD_NAME"

# Attempt to mount the DMG file and capture the output
echo "Attempting to mount DMG file: $SOURCE_DMG"
MOUNT_OUTPUT=$(hdiutil attach "$SOURCE_DMG" 2>&1)

# Capture the exit status of the mount command
MOUNT_STATUS=$?

echo "$MOUNT_OUTPUT"

if [ $MOUNT_STATUS -ne 0 ]; then
    echo "Error: Failed to mount the DMG file. Exiting."
    exit 1
else
    MOUNT_DIR=$(echo "$MOUNT_OUTPUT" | grep -o '/Volumes/.*$')
    echo "Success: Mounted at $MOUNT_DIR"
fi

# Ensure the destination directory exists or create it
if [ ! -d "$BUILD_PATH" ]; then
    echo "Destination directory not found. Creating path: $BUILD_PATH"
    mkdir -p "$BUILD_PATH"
    if [ $? -ne 0 ]; then
      echo "Error: Failed to create destination directory. Exiting."
      exit 1
    else
      echo "Successfully created destination directory."
    fi
else
    echo "Destination directory already exists. Continuing."
fi

# Copy files to the destination
echo "Copying Overmind binary to ${BUILD_PATH}/${BUILD_NAME}-darwin"
cp -R "${MOUNT_DIR}/overmind" "${BUILD_PATH}/${BUILD_NAME}-darwin"
if [ $? -ne 0 ]; then
  echo "Error: Failed to copy files. Exiting."
  hdiutil detach "$(echo $MOUNT_DIR | sed 's/ /\\ /g')"
  exit 1
else
  echo "Successfully copied Overmind binary."
fi

# Attempt to unmount the DMG and capture the output
echo "Attempting to unmount DMG at $MOUNT_DIR"
UNMOUNT_OUTPUT=$(hdiutil detach "$(echo $MOUNT_DIR | sed 's/ /\\ /g')" 2>&1)
UNMOUNT_STATUS=$?

echo "$UNMOUNT_OUTPUT"

if [ $UNMOUNT_STATUS -ne 0 ]; then
  echo "Warning: Failed to unmount DMG cleanly. Manual cleanup might be required."
else
  echo "DMG unmounted successfully."
fi

echo "Process complete. Exiting."