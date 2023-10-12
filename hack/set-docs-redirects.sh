#!/usr/bin/env bash

# ApplicationSet docs now live at https://argo-cd.readthedocs.io/en/latest/operator-manual/applicationset/
# This script adds redirects to the top of each ApplicationSet doc to redirect to the new location.

set -e pipefail

new_docs_base_path="https://argo-cd.readthedocs.io/en/latest/operator-manual/applicationset/"
new_docs_base_path_regex=$(echo "$new_docs_base_path" | sed 's/\//\\\//g')

# Loop over files in the docs directory recursively. For each file, use sed to add the following redirect to the top:
# <meta http-equiv="refresh" content="0; url='https://argo-cd.readthedocs.io/en/latest/operator-manual/applicationset/{FILE PATH}'" />
# FILE_PATH should be the path to the file relative to the docs directory, stripped of the .md extension.

files=$(find docs -type f -name '*.md')
for file in $files; do
  file_path=$(echo "$file" | sed 's/^docs\///' | sed 's/\.md$/\//')
  echo "Adding redirect to $file_path"
  # If a redirect is already present at the top of the file, remove it.
  sed '1s/<meta http-equiv="refresh" content="0; url='\''https:\/\/argo-cd.readthedocs.io\/en\/latest\/operator-manual\/applicationset\/.*'\'' \/>//' "$file" > "$file.tmp"
  mv "$file.tmp" "$file"

  # Add the new redirect.
  # Default to an empty path.
  file_path_plain=""
  file_path_regex=""
  if curl -s -o /dev/null -w "%{http_code}" "$new_docs_base_path$file_path" | grep -q 200; then
    # If the destination path exists, use it.
    file_path_plain="$file_path/"
    file_path_regex=$(echo "$file_path" | sed 's/\//\\\//g')
  else
    echo "WARNING: $new_docs_base_path$file_path does not exist. Using empty path."
  fi

  notice="!!! important \"This page has moved\"\n    This page has moved to [$new_docs_base_path$file_path_plain]($new_docs_base_path$file_path_plain). Redirecting to the new page.\n"

  notice_regex=$(echo "$notice" | sed 's/\//\\\//g')

  sed "1s/^/<meta http-equiv=\"refresh\" content=\"1; url='$new_docs_base_path_regex$file_path_regex'\" \/>\\n\\n$notice_regex\\n/" "$file" > "$file.tmp"
  mv "$file.tmp" "$file"
done
