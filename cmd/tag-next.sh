#!/bin/bash

set -e

usage() {
    echo "Usage: $0 [-M|-m|-p]"
    echo "  -M    Increment major version"
    echo "  -m    Increment minor version (default)"
    echo "  -p    Increment patch version"
    exit 1
}

# Default: Increment minor
INCREMENT="minor"

# Parse flags
while getopts "Mmp" opt; do
    case $opt in
        M) INCREMENT="major" ;;
        m) INCREMENT="minor" ;;
        p) INCREMENT="patch" ;;
        *) usage ;;
    esac
done
shift $((OPTIND - 1))

# Fetch tags from the remote repository
git fetch --tags

# Get the latest tag by listing all tags and sorting them
TAG=$(git tag | sort -V | tail -n 1)

# Ensure a tag exists
if [ -z "$TAG" ]; then
    echo "Error: No existing tags found. Please create an initial tag."
    exit 1
fi

# Extract the version components
IFS='.' read -ra PARTS <<< "$TAG"

# Ensure version format is correct
if [ ${#PARTS[@]} -ne 3 ]; then
    echo "Error: Tag format should be X.Y.Z"
    exit 1
fi

# Determine next version
if [ "$INCREMENT" == "major" ]; then
    NEXT_TAG="$((PARTS[0] + 1)).0.0"
elif [ "$INCREMENT" == "minor" ]; then
    NEXT_TAG="${PARTS[0]}.$((PARTS[1] + 1)).0"
else
    NEXT_TAG="${PARTS[0]}.${PARTS[1]}.$((PARTS[2] + 1))"
fi

# Tag and push the new tag to the origin repository
echo "Tagging and pushing $NEXT_TAG"

git tag "$NEXT_TAG"
git push origin "$NEXT_TAG"
