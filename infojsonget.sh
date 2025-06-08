#!/usr/bin/env bash

# Check if a URL is provided as an argument
if [ -z "$1" ]; then
  echo "Usage: $0 <YouTube URL> <optional filename pattern>"
  exit 1
fi

if [ -z "$2" ]; then
  pattern="%(id) - %(title).json"
else
  pattern="$2"
fi

html_content=$(curl -s "$1")

video_id=$(echo "$html_content" | grep -oP '(?<=<meta property="og:url" content="https://www.youtube.com/watch\?v=)[^&"]+')

title=$(echo "$html_content" | grep -oP '(?<=<meta property="og:title" content=").*?(?=")' | recode html..utf8) 

thumbnail=$(echo "$html_content" | grep -oP '(?<=<meta property="og:image" content=").*?(?=")')

description=$(echo "$html_content" | grep -oP '(?<=<meta property="og:description" content=").*?(?=")' | recode html..utf8)

#channel=$(echo "$html_content" | grep -oP '(?<=<meta property="og:site_name" content=").*?(?=")')
#channel=$(echo "$html_content" | grep -oP '(?<=<link itemprop="url" href="http://www.youtube.com/@).*?(?=")')
#channel=$(echo "$html_content" | grep -oP '(?<=<link itemprop="name" href="http://www.youtube.com/@).*?(?=")')
channel=$(echo "$html_content" | grep -oP '(?<=<link itemprop="name" content=").*?(?=")' | recode html..utf8)

upload_date=$(echo "$html_content" | grep -oP '(?<=<meta itemprop="uploadDate" content=").*?(?=")')

formatted_upload_date=$(echo "$upload_date" | sed 's/-//g' | cut -c1-8)

extractor_key="Youtube"


# Use the provided or default pattern to generate the filename
# Replace '%(id)s' with video_id and '%(title)s' with title in the pattern

cleaned_title=$(echo "$title" | sed 's/[\/:*?"<>|]/_/g')

filename=$(echo "$pattern" | sed "s/%(id)/$video_id/" | sed "s/%(title)/$cleaned_title/")

# Clean up filename (remove or replace any invalid characters for filenames, e.g., slashes)
#filename=$(echo "$filename" | sed 's/[\/:*?"<>|]/_/g')

json_content=$(cat <<EOF
{
  "id": "$video_id",
  "title": "$title",
  "thumbnail": "$thumbnail",
  "description": "$description",
  "categories": [],
  "tags": [],
  "channel": "$channel",
  "uploader": "$channel",
  "upload_date": "$formatted_upload_date",
  "extractor_key": "$extractor_key"
}
EOF
)

echo "$json_content" > "$filename"
