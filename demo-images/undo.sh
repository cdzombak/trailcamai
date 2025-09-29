#!/bin/bash

# undo.sh - Move all files from subfolders back to demo-images and remove empty subfolders
# Usage: ./undo.sh (from demo-images directory)
#        OR: ./demo-images/undo.sh (from repo root)

set -e  # Exit on any error

# Determine the correct working directory
if [[ "$(basename "$(pwd)")" == "demo-images" ]]; then
    # Running from within demo-images directory
    DEMO_DIR="$(pwd)"
elif [[ -d "demo-images" ]]; then
    # Running from repo root or parent directory
    DEMO_DIR="$(pwd)/demo-images"
else
    echo "Error: Cannot find demo-images directory"
    echo "Please run this script from:"
    echo "  - The demo-images directory: ./undo.sh"
    echo "  - The repo root: ./demo-images/undo.sh"
    exit 1
fi

cd "$DEMO_DIR"
echo "Undoing trailcamai organization in $DEMO_DIR..."

# Find all subdirectories (excluding hidden ones and the script itself)
subdirs=$(find . -maxdepth 1 -type d ! -name "." ! -name ".*")

if [ -z "$subdirs" ]; then
    echo "No subdirectories found. Nothing to undo."
    exit 0
fi

files_moved=0

# Move files from each subdirectory back to current directory
for subdir in $subdirs; do
    subdir_name=$(basename "$subdir")
    echo "Processing subdirectory: $subdir_name"

    # Find all files in this subdirectory (excluding the undo script itself)
    files=$(find "$subdir" -type f ! -name "undo.sh")

    if [ -n "$files" ]; then
        file_count=$(echo "$files" | wc -l)
        echo "  Moving $file_count file(s) from $subdir_name/"

        # Move each file to the parent directory
        while IFS= read -r file; do
            filename=$(basename "$file")
            if [ -f "$filename" ]; then
                echo "    Warning: $filename already exists in target directory, skipping"
            else
                mv "$file" .
                echo "    Moved $filename"
                ((files_moved++))
            fi
        done <<< "$files"
    else
        echo "  No files found in $subdir_name/"
    fi
done

# Remove empty subdirectories
echo "Removing empty subdirectories..."
dirs_removed=0
for subdir in $subdirs; do
    subdir_name=$(basename "$subdir")
    if [ -d "$subdir" ]; then
        # Check if directory is empty (ignoring hidden files)
        if [ -z "$(find "$subdir" -mindepth 1 -not -name ".*" -print -quit)" ]; then
            rmdir "$subdir"
            echo "  Removed empty directory: $subdir_name/"
            ((dirs_removed++))
        else
            echo "  Warning: $subdir_name/ is not empty, skipping removal"
            echo "    Remaining contents:"
            ls -la "$subdir" | grep -v "^total" | sed 's/^/      /'
        fi
    fi
done

echo ""
echo "Undo complete!"
echo "  Files moved: $files_moved"
echo "  Directories removed: $dirs_removed"
echo ""
echo "Current demo-images directory contents:"
ls -la | grep -v "^total" | sed 's/^/  /'