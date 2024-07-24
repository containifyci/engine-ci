#!/bin/bash
REPO="containifyci/$1"
curl -H "Accept: application/vnd.github+json" \
  -H "Authorization: token $(gh auth token)" \
  https://api.github.com/repos/${REPO}/actions/caches
cache_ids=$(curl -H "Accept: application/vnd.github+json" -H "Authorization: token $(gh auth token)" https://api.github.com/repos/$REPO/actions/caches | jq -r '.actions_caches[].id')

for cache_id in $cache_ids; do
  curl -X DELETE -H "Accept: application/vnd.github+json" -H "Authorization: token $(gh auth token)" https://api.github.com/repos/$REPO/actions/caches/$cache_id
done
