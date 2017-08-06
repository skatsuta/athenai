#!/usr/bin/env bash
set -eu

read -p "What is a new version number? " version

read -p "Have you updated the version number in the source and CHANGELOG? [y/N] " yn
while true; do
  case $yn in
    [Yy]*)
      break
      ;;
    [Nn]*|'')
      echo 'Please update README before releasing a new version.'
      exit 1
      ;;
    *)
      echo "$yn: didn't match anything"
  esac
done

set -x
git tag -a $version
git push --tags
goreleaser --rm-dist
