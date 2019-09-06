#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if [[ -z $FILES ]]; then
    echo 'ERROR no $FILE specified'
    exit 1
fi

if [[ -z $CIRCLE_TAG]]; then
    tag="${CIRCLE_SHA1:0:7}"
    name="${CIRCLE_SHA1:0:7}"
else
    tag="${CIRCLE_TAG}"
    name="${CIRCLE_TAG}"
fi

upload_url=$(curl -Ssf -d"
{
    \"tag_name\": \"$tag\",
    \"target_commitish\": \"${CIRCLE_BRANCH:-master}\",
    \"name\": \"$name\",
    \"body\": \"$RELEASE_BODY\",
    \"draft\": ${DRAFT:-true},
    \"prerelease\": false
}" https://api.github.com/repos/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}/releases?access_token=$GH_API_TOKEN \
| jq -r .upload_url | cut -d'{' -f1)

sleep 2

for file in $FILES; do
    echo "uploading $file ..."
    curl -Ssf \
        -XPOST \
        --data-binary "@${file}" \
        -H 'Content-Type: application/zip' \
        -H "Authorization: token $GH_API_TOKEN" \
        "${upload_url}?name=$(basename $file)" | jq .
done