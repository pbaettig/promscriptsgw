#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if [[ -z $FILES ]]; then
    echo 'ERROR no $FILE specified'
    exit 1
fi

echo "
{
    \"tag_name\": \"$TAG_NAME\",
    \"target_commitish\": \"${CIRCLECI_BRANCH:-master}\",
    \"name\": \"$TAG_NAME\",
    \"body\": \"$RELEASE_BODY\",
    \"draft\": ${DRAFT:-true},
    \"prerelease\": false
}"

upload_url=$(curl -Ssf -d"
{
    \"tag_name\": \"$TAG_NAME\",
    \"target_commitish\": \"${CIRCLECI_BRANCH:-master}\",
    \"name\": \"$TAG_NAME\",
    \"body\": \"$RELEASE_BODY\",
    \"draft\": ${DRAFT:-true},
    \"prerelease\": false
}" https://api.github.com/repos/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}/releases?access_token=$GH_API_TOKEN \
| jq -r .upload_url | cut -d'{' -f1)

sleep 2
export IFS=";"

for file in $FILES; do
    curl -Ssf \
        -XPOST \
        --data-binary "@${file}" \
        -H 'Content-Type: application/zip' \
        -H "Authorization: token $GH_API_TOKEN" \
        "${upload_url}?name=$(basename $file)" | jq .
done