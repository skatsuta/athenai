# CHANGELOG

## 0.2.0

* Adds `--encrypt/-e` and `--kms/-k` flags to `run` command.
  * You can now encrypt query results in Amazon S3 with `SSE_S3`, `SSE_KMS` or `CSE_KMS` encryption option.
* Builds binary with Go 1.9.

## 0.1.2

* N/A.

## 0.1.1

* Fixes an issue where `index out of range` panic occurs on rare occasions.

## 0.1.0

* Initial release.
