#!/bin/bash

echo "---"
grep --color=never --no-filename "image:" "${1}"/templates/* | sed -e "s/^[ \t]*image: /- /"  | uniq | sort
