#!/bin/bash

set -euf

THRESHOLD_MONTHS='3'
TAG_PATTERN='[0-9]+\.[0-9]+\.[0-9]+-dev(\.[0-9]+)?'

tags=($(git for-each-ref --format='%(creatordate:unix);%(refname:short)' refs/tags | grep -E "^[0-9]{10};${TAG_PATTERN}"))
threshold="$(date --date="${THRESHOLD_MONTHS} months ago" '+%s')"
now="$(date +%s)"
delete=()

echo "Tags older than ${THRESHOLD_MONTHS} months:"
for tag in "${tags[@]}"; do
    ref_time="$(echo "${tag}" | cut -d ';' -f 1 )"
    ref_name="$(echo "${tag}" | cut -d ';' -f 2 )"

    if [[ ${ref_time} < ${threshold} ]]; then
        delete+=("${ref_name}")
        diff=$(( (${now} - ${ref_time}) / 60 / 60 / 24 ))
        echo "${ref_name} (${diff} days)"
    fi
done

if [ "${#delete[@]}" -gt 0 ]; then
    #git push --delete origin ${delete[@]}
else
    echo 'Nothing to remove.'
fi
