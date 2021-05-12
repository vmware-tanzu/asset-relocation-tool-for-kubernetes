#!/bin/bash

echo "---"
find "${1}/templates" -type f -exec grep --color=never --no-filename "image:" {} \; | sed -e "s/^[ \t]*image: /- /" | sort | uniq
