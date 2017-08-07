#!/usr/bin/env bash
set -eu

read -p "What is a new version number? " version

read -p "Have you updated the version number in the source code and CHANGELOG.md? [y/N] " yn
while true; do
  case $yn in
    [Yy]*)
      break
      ;;
    [Nn]*|'')
      echo 'Please update them before releasing a new version.'
      exit 1
      ;;
    *)
      echo "Please enter y or n."
  esac
done

set -x
git tag -a $version
git push origin master
git push --tags
goreleaser --rm-dist
