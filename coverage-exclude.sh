#!/bin/bash

# Detect if running on macOS or Linux (Ubuntu)
if sed --version >/dev/null 2>&1; then
    SED_COMMAND="sed -i"  # GNU sed (Linux)
else
    SED_COMMAND="sed -i ''"  # BSD/macOS sed
fi

while read -r p || [ -n "$p" ]
do
    escaped_p=$(echo "$p" | sed 's|/|\\/|g')
    eval "$SED_COMMAND \"/$escaped_p/d\" ./coverage.out"
done < ./coverage-exclude.txt
