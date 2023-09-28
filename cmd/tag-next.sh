#!/bin/bash

set -e

# Fetch tags from the remote repository
git fetch --tags

# Get the latest tag by listing all tags and sorting them
TAG=$(git tag | sort -V | tail -n 1)

# Extract the version components
IFS='.' read -ra PARTS <<< "$TAG"

# Increment the last component of the tag
NEXT_PART=$((PARTS[2] + 1))
NEXT_TAG="${PARTS[0]}.${PARTS[1]}.$NEXT_PART"

# Tag and push the new tag to the origin repository
echo "Tagging and pushing $NEXT_TAG"
git tag "$NEXT_TAG"
git push origin master "$NEXT_TAG"
