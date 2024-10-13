#!/usr/bin/env bash

# Check if a URL is provided as an argument
if [ -z "$1" ]; then
  echo "Usage: $0 <YouTube URL>"
  exit 1
fi

html_content=$(curl -s "$1")

video_id=$(echo "$html_content" | grep -oP '(?<=<meta property="og:url" content="https://www.youtube.com/watch\?v=)[^&"]+')

title=$(echo "$html_content" | grep -oP '(?<=<meta property="og:title" content=").*?(?=")')

thumbnail=$(echo "$html_content" | grep -oP '(?<=<meta property="og:image" content=").*?(?=")')

description=$(echo "$html_content" | grep -oP '(?<=<meta property="og:description" content=").*?(?=")')

#channel=$(echo "$html_content" | grep -oP '(?<=<meta property="og:site_name" content=").*?(?=")')
#channel=$(echo "$html_content" | grep -oP '(?<=<link itemprop="url" href="http://www.youtube.com/@).*?(?=")')
#channel=$(echo "$html_content" | grep -oP '(?<=<link itemprop="name" href="http://www.youtube.com/@).*?(?=")')
channel=$(echo "$html_content" | grep -oP '(?<=<link itemprop="name" content=").*?(?=")')

upload_date=$(echo "$html_content" | grep -oP '(?<=<meta itemprop="uploadDate" content=").*?(?=")')

formatted_upload_date=$(echo "$upload_date" | sed 's/-//g' | cut -c1-8)

extractor_key="Youtube"

echo '{
  "id": "'$video_id'",
  "title": "'$title'",
  "thumbnail": "'$thumbnail'",
  "description": "'$description'",
  "categories": [
  ],
  "tags": [
  ],
  "channel": "'$channel'",
  "uploader": "'$channel'",
  "upload_date": "'$formatted_upload_date'",
  "extractor_key": "'$extractor_key'"
}'