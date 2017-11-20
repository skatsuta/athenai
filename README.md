<p align="center"><img src="docs/logo.jpg" alt="logo"></p>


# Athenai

[![wercker status](https://app.wercker.com/status/554177081896c1c508e365ffa1c8cc26/s/master "wercker status")](https://app.wercker.com/project/byKey/554177081896c1c508e365ffa1c8cc26)
[![codecov](https://codecov.io/gh/skatsuta/athenai/branch/master/graph/badge.svg?token=dkRnWrYYa9)](https://codecov.io/gh/skatsuta/athenai)
[![Go Report Card](https://goreportcard.com/badge/github.com/skatsuta/athenai)](https://goreportcard.com/report/github.com/skatsuta/athenai)
[![Readme Score](http://readme-score-api.herokuapp.com/score.svg?url=skatsuta/athenai)](http://clayallsopp.github.io/readme-score?url=skatsuta/athenai)

Have fun with Amazon Athena from command line! 🕊


## Overview

Athenai is a simple and easy-to-use command line tool that runs SQL statements on [Amazon Athena](https://aws.amazon.com/athena/).

With Athenai you can run multiple queries easily at a time on Amazon Athena and can see the results in table or CSV format interactively once the executions have finished.

"A picture is worth a thousand words." See the **[Demo](#demo)** section to see how it works 👀


## Demo

![Demo](docs/demo.gif)


## Features

- **Easy to use**: provide queries, wait for query executions and see the results once the executions have finished.
- **Various input methods**: REPL, command line arguments or SQL file.
- **Concurrent executions**: run multiple queries concurrently at a time.
- **Query cancellation**: cancel queries if `Ctrl-C` is pressed during the executions.


## Installation

Athenai is currently supported on macOS and Linux.

### Installing from binary

[Download the archive](https://github.com/skatsuta/athenai/releases/latest), extract it and place the executable somewhere in `PATH`.
For example,

```bash
# Please replace ${VERSION}, ${OS} and ${ARCH} with appropriate values for your platform.
$ curl -sL https://github.com/skatsuta/athenai/releases/download/${VERSION}/athenai_${OS}_${ARCH}.tar.gz -o athenai.tar.gz
$ tar -xzf athenai.tar.gz
$ mv athenai /usr/local/bin/ # or wherever you like in PATH
$ athenai --help
```

### Installing via Homebrew (macOS only)

Athenai provides [a repository for Homebrew](https://github.com/skatsuta/homebrew-athenai). If you use [Homebrew](https://brew.sh/), you can install the binary as follows:

```bash
$ brew install skatsuta/athenai/athenai
$ athenai --help
```

### Installing from source

If you use Go, you can build and install your binary from source using `go get`:

```bash
$ go get -v github.com/skatsuta/athenai
# ...
$ $GOPATH/bin/athenai --help
```


## Setup

### AWS creadentitals (Required)

Before using this tool, you need to set up AWS credentials just like using AWS CLI or AWS SDK.
If you already use them and have sufficient IAM permissions to use Amazon Athena, you may not need this step 😊
In that case just skip to the next **[Default configuration file](#default-configuration-file-optional-)** section! 🚀

To set up AWS credentials, there are mainly three ways:

* [Configuring environment variables](http://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html): simple, but not recommended.
* [Configuring shared credentials file (`$HOME/.aws/credentials`)](http://docs.aws.amazon.com/cli/latest/userguide/cli-config-files.html): recommended for use on local computers.
* [Configuring instance profile](http://docs.aws.amazon.com/cli/latest/userguide/cli-metadata.html): recommended for use on EC2 instances.

Please follow one of the above instructions corresponding to your use case.

After setting it up, make sure your IAM user or role to use has correct Amazon Athena and Amazon S3 permissions.
**To use Athenai, [AmazonAthenaFullAccess Managed Policy](http://docs.aws.amazon.com/athena/latest/ug/access.html#amazonathenafullaccess-managed-policy-contents) has to be attached to your IAM user or role to use.**

### Default configuration file (Optional; Recommended)

You can optionally set your default configuration values into your `$HOME/.athenai/config` file to simplify every command execution.

Write the following into `$HOME/.athenai/config` and save it.<br>
(Modify the `profile` value if you use another profile in your `$HOME/.aws/credentials`, instead of `default`.)

```ini
[default]
# (Optional) Profile in your $HOME/.aws/credentials file
profile = default
# AWS region
region = us-east-1
# Database name
database = sampledb
# Output location in S3 where query results will be stored
location = s3://aws-athena-query-results-<YOUR_ACCOUNT_ID>-us-east-1/
```

Then Athenai loads the configuration file automatically and you can omit the option flags when running commands.

See the **[Configuration file](#configuration-file)** section for more details.


## Usage

#### Note: option flags

In this section, it is assumed that you have already set up [the default configuration file](#default-configuration-file-optional-recommended), and option flags are omitted to describe the main usage as simply as possible.
If you haven't done it yet or want to override the default values in your config file, run a command with flags like this:

```
$ athenai run \
  --profile default \
  --region us-east-1 \
  --database sampledb \
  --location s3://aws-athena-query-results-<YOUR_ACCOUNT_ID>-us-east-1/ \
  "SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;"
```

(Modify the `--profile` flag if you use another profile in your `$HOME/.aws/credentials`, instead of `default`.)

### Running queries interactively (REPL mode)

![Running queries interactively](docs/run_repl.gif)

To run queries on interactive (REPL) mode, run `athenai run` command with no arguments except for flags:

```
$ athenai run
athenai> SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
⠚ Running query...
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 3.12 seconds | Data scanned: 101.27 KB
Location: s3://aws-athenai-demo/835573f3-8ff9-4fd3-a1a4-097f4921b93d.csv
athenai> SHOW DATABASES; SHOW TABLES;
⠴ Running query...
Query: SHOW DATABASES;
+-----------------+
| cloud_trail     |
| cloudfront_logs |
| default         |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.32 seconds | Data scanned: 0 B
Location: s3://aws-athenai-demo/ba7de01a-8159-4a66-8453-d68294d9871d.txt

Query: SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.40 seconds | Data scanned: 0 B
Location: s3://aws-athenai-demo/150a8494-0750-43ad-8971-9dd68c75f076.txt
athenai> ^D
$ 
```

In REPL mode you can use common key shortcuts just like on most shells. For example,

Key | Action
:---:|---
`↑`/`Ctrl-P` | Move to the previous line in history
`↓`/`Ctrl-N` | Move to the next line in history
`Ctrl-A` | Move cursor to the beginning of the line
`Ctrl-E` | Move cursor to the end of the line
`Ctrl-H` | Delete a character

and so on.
Available shortcuts are listed [here](https://github.com/chzyer/readline/blob/master/doc/shortcut.md).

Your query history is saved to the `$HOME/.athenai/history` file automatically.

To exit REPL, press `Ctrl-C` or `Ctrl-D` on empty line.

### Running queries from command line arguments

![Running queries from command line arguments](docs/run_arg.gif)

To run queries from command line arguments, just pass them to `athenai run` command:

```
$ athenai run "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠙ Running query...
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.35 seconds | Data scanned: 101.27 KB
Location: s3://aws-athenai-demo/71940569-f9ce-41d7-81c0-587ff0aeffda.csv

$ athenai run "SHOW DATABASES; SHOW TABLES;"
⠦ Running query...
Query: SHOW DATABASES;
+-----------------+
| cloud_trail     |
| cloudfront_logs |
| default         |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.38 seconds | Data scanned: 0 B
Location: s3://aws-athenai-demo/c572a884-1cba-472d-a568-ac8e1e75551b.txt

Query: SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.41 seconds | Data scanned: 0 B
Location: s3://aws-athenai-demo/e70c473c-9994-4247-991c-8d739a7249cc.txt
```

### Running queries from SQL file

![Running queries from an SQL file](docs/run_file.gif)

To run queries from an SQL file, pass its path with `file://` prefix to `athenai run` command:

```
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
$ athenai run file://sample.sql
⠚ Running query...
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-08-05 | 15:56:57 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4257 | 10.0.0.3  | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4251 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:59 |    10 | 10.0.0.15 | GET    |    304 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.03 seconds | Data scanned: 101.82 KB
Location: s3://aws-athenai-demo/836b5b3f-5fdd-447f-9b02-dc869bc8d03d.csv
```

or pass its contents via stdin:

```
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
$ athenai run < sample.sql
⠲ Running query...
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 1.99 seconds | Data scanned: 101.27 KB
Location: s3://aws-athenai-demo/686f3498-cb31-4731-84ed-5dce9614c6c3.csv
```

### Running DDL statements to manipulate metadata

![Running CREATE statements to create a database and table](docs/run_ddl.gif)

Athenai supports not only `SELECT`, `SHOW` and `DESCRIBE` statements, but also DDL statements such as `CREATE`, `ALTER` and `DROP`.
You can run any available DDL statements listed [here](http://docs.aws.amazon.com/athena/latest/ug/language-reference.html) to manipulate metadata.

Since these statements usually show no results, the outputs of them are `(No output)` with query info, like the following:

```
$ athenai run "CREATE DATABASE testdb"
⠳ Running query...
Query: CREATE DATABASE testdb;
(No output)
Run time: 0.41 seconds | Data scanned: 0 B
Location: s3://aws-athenai-demo/5fb06aaf-4123-49d0-b7fb-5876e99d788e.txt

$ athenai run < /tmp/create_table.sql
⠦ Running query...
Query: CREATE EXTERNAL TABLE IF NOT EXISTS testdb.persons (
  id INT,
  name STRING,
  age INT
)
ROW FORMAT SERDE 'org.apache.hadoop.hive.serde2.OpenCSVSerde'
WITH SERDEPROPERTIES (
  'separatorChar' = ',',
  'quoteChar' = '\"',
  'escapeChar' = '\\'
)
STORED AS TEXTFILE
LOCATION 's3://aws-athenai-demo/csv/';
(No output)
Run time: 0.95 seconds | Data scanned: 0 B
Location: s3://aws-athenai-demo/184344f3-3f6d-4fb9-9716-a89a7eb32ab6.txt

$ athenai run "SELECT * FROM testdb.persons"
⠴ Running query...
Query: SELECT * FROM testdb.persons;
+----+---------+-----+
| id | name    | age |
|  1 | alice   |  20 |
|  2 | bob     |  30 |
|  3 | charlie |  40 |
+----+---------+-----+
Run time: 1.27 seconds | Data scanned: 51 B
Location: s3://aws-athenai-demo/36db3707-c0d7-416f-99af-aec3d6360583.csv
```

### Running multiple statements sequentially

![Running multiple statements sequantially](docs/run_seq.gif)

When you run multiple statements with Athenai, it runs up to 5 statements concurrently at a time by default, and subsequent statements are executed once prior ones have finished.

Sometimes, however, you may need to run each statement sequentially.
For example, suppose you are going to run the following 3 statements (`CREATE DATABASE` => `CREATE TABLE` => `SELECT`):

##### sample.sql

```sql
CREATE DATABASE IF NOT EXISTS testdb;

CREATE EXTERNAL TABLE IF NOT EXISTS testdb.persons (
  id INT,
  name STRING,
  age INT
)
ROW FORMAT SERDE 'org.apache.hadoop.hive.serde2.OpenCSVSerde'
WITH SERDEPROPERTIES (
  'separatorChar' = ',',
  'quoteChar' = '\"',
  'escapeChar' = '\\'
)
STORED AS TEXTFILE
LOCATION 's3://aws-athenai-demo/csv/';

SELECT * FROM testdb.persons;
```

In these statements the second and third one depend on the previous one of each respectively, so they cannot be executed concurrently and need to be run sequentially.

In this case, you can specify the maximum number of concurrent query executions by using `--concurrent/-c` flag.
To run multiple statements sequentially, specify `--concurrent 1`:

```
$ athenai run --concurrent 1 < sample.sql
```

This command runs each statement sequentially and you should get the results you expect! 😄

#### Caution

Althrough it is possible for you to specify max concurrency to more than 5 with `--concurrent/-c` flag, usually it is not recommended because the default concurrency limits are 5 concurrent DDL and SELECT statements at a time, as described in [Service Limits of Amazon Athena](http://docs.aws.amazon.com/athena/latest/ug/service-limits.html).

There is no problem if you have requested a limit increase for the limit, however 😉

### Encrypting query results in Amazon S3

You can encrypt query results in Amazon S3 by running queries with `--encrypt/-e` flag.
The following encryption types are currently available.

* [Amazon S3 server-side encryption with Amazon S3-managed keys](https://docs.aws.amazon.com/AmazonS3/latest/dev/UsingServerSideEncryption.html) (**`SSE_S3`**)
* [Server-side encryption with KMS-managed keys](https://docs.aws.amazon.com/AmazonS3/latest/dev/UsingKMSEncryption.html) (**`SSE_KMS`**)
* [Client-side encryption with KMS-managed keys](https://docs.aws.amazon.com/AmazonS3/latest/dev/UsingClientSideEncryption.html) (**`CSE_KMS`**)

If you use `SSE_KMS` or `CSE_KMS`, you also need to provide your KMS key ARN or ID using `--kms/-k` flag.

See [EncryptionConfiguration data type on Amazon Athena API Reference](http://docs.aws.amazon.com/athena/latest/APIReference/API_EncryptionConfiguration.html) for more details about parameters for encryption configuration.

##### Using Amazon S3 server-side encryption with Amazon S3-managed keys (`SSE_S3`)

```
$ athenai run --encrypt SSE_S3 ...
```

##### Using server-side encryption with KMS-managed keys (`SSE_KMS`)

```
$ athenai run --encrypt SSE_KMS --kms $YOUR_KMS_KEY_ARN_OR_ID ...
```

##### Using client-side encryption with KMS-managed keys (`CSE_KMS`)

```
$ athenai run --encrypt CSE_KMS --kms $YOUR_KMS_KEY_ARN_OR_ID ...
```

#### Note

If you want to make every query result executed by Athenai encrypted, I recommend you add these encryption configurations to your `$HOME/.athenai/config` file.
For example, to use `SSE_KMS` encryption type, add these lines into your `default` section:

```ini
encrypt = SSE_KMS
kms = <YOUR_KMS_KEY_ARN_OR_ID_HERE>
```

It eliminates the need of specifying the encryption flags every time and ensures your every query result will be encrypted with `SSE_KMS`.

### Canceling queries

![Canceling queries](docs/run_cancel.gif)

You can cancel queries by pressing `Ctrl-C` during the query executions.

```
$ athenai run "SELECT * FROM sampledb.cloudfront_logs"   # Oops! Full scan by mistake!
⠖ Running query... ^C   # Press Ctrl-C
⠋ Canceling...
$ # Whew! That was close.
```

Athenai calls [StopQueryExecution API](http://docs.aws.amazon.com/athena/latest/APIReference/API_StopQueryExecution.html) to stop the query executions once `Ctrl-C` is pressed, so charges for the queries should stop too.

### Showing results of completed query executions

![Showing results of completed query executions](docs/show.gif)


You can show the results of completed query executions without re-running the same queries.

Run the command below:

```
$ athenai show
```

and select query executions you want to show from the list with interactive filtering:

```
QUERY>                                                                                                                                                IgnoreCase [48 (1/1)]
2017-07-26 14:11:36 +0000 UTC   SHOW TABLES SUCCEEDED   0.37 seconds    0 B
2017-07-26 14:11:36 +0000 UTC   SELECT timestamp, requestip, backendip FROM elb_logs LIMIT 3   SUCCEEDED   0.55 seconds    17.80 KB
2017-07-26 14:11:36 +0000 UTC   SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 10   SUCCEEDED   2.23 seconds    101.27 KB
2017-07-26 14:11:36 +0000 UTC   SHOW DATABASES  SUCCEEDED   0.38 seconds    0 B
(snip)
```

Athenai uses [peco/peco](https://github.com/peco/peco) as a library that performs interactive filtering.
Frequently used key mappings are as follows:

Key | Action
:---:|---
`↑`/`Ctrl-P` | Move up
`↓`/`Ctrl-N` | Move down
`Ctrl-Space` | Select/Unselect each entry
`Ctrl-A` | Move cursor to the beginning of the line
`Ctrl-E` | Move cursor to the end of the line
`Ctrl-H` | Delete a character

Available key mappings are listed [here](https://github.com/peco/peco#default-keymap).
You can select multiple entries by pressing `Ctrl-Space` on each entry.


After you have selected the entries to show, hit `Enter` and you will see the results of selected query executions like the following:

```
$ athenai show
⠋ Loading history...
⠚ Fetching results...
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-08-05 | 15:56:57 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4257 | 10.0.0.3  | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4251 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:59 |    10 | 10.0.0.15 | GET    |    304 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.03 seconds | Data scanned: 101.82 KB
Location: s3://aws-athenai-demo/836b5b3f-5fdd-447f-9b02-dc869bc8d03d.csv
⠞ Fetching results...
Query: SHOW DATABASES;
+-----------------+
| cloud_trail     |
| cloudfront_logs |
| default         |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.38 seconds | Data scanned: 0 B
Location: s3://aws-athenai-demo/c572a884-1cba-472d-a568-ac8e1e75551b.txt
```

By default the `show` command lists up to the latest 50 query executions except for ones in `FAILED` state.
You can configure the number by specifying `--count/-c` flag:

```
$ athenai show --count 100   # Lists up to the latest 100 query executions
```

If you want to list all of your completed query executions, specify `0`:

```
$ athenai show --count 0
```

Note that `athenai show --count 0` may be very slow depending on the total number of your query executions.

### Printing results in CSV format

![Printing results in CSV format](docs/format_csv.gif)

If you want to print query results in CSV format instead of table format, specify `--format/-f csv` flag.

```
$ athenai run --format csv "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠲ Running query...
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
date,time,bytes,requestip,method,status
2014-07-05,15:00:00,4260,10.0.0.15,GET,200
2014-07-05,15:00:00,10,10.0.0.15,GET,304
2014-07-05,15:00:00,4252,10.0.0.15,GET,200
2014-07-05,15:00:00,4257,10.0.0.8,GET,200
2014-07-05,15:00:03,4261,10.0.0.15,GET,200
Run time: 2.20 seconds | Data scanned: 101.27 KB
Location: s3://aws-athenai-demo/ad90ad38-15fe-4f61-9c0d-2a648bb2f8f3.csv
```

You can also use this flag with `athenai show` command.

```
$ athenai show --format csv
⠳ Loading history...
⠚ Fetching results...
QueryExecutionId: 7226c8a5-c3b6-4399-97fb-ea7683e774d1
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
date,time,bytes,requestip,method,status
2014-07-05,15:00:00,4260,10.0.0.15,GET,200
2014-07-05,15:00:00,10,10.0.0.15,GET,304
2014-07-05,15:00:00,4252,10.0.0.15,GET,200
2014-07-05,15:00:00,4257,10.0.0.8,GET,200
2014-07-05,15:00:03,4261,10.0.0.15,GET,200
Run time: 1.90 seconds | Data scanned: 101.27 KB
```

### Outputting (Saving) results to file

![Outputting (Saving) results to a file](docs/run_output.gif)

If you want to output (save) the query results to a file, use `--output/-o` flag to specify the file path:

```
$ athenai run --output /tmp/results.txt "SELECT date, time, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠙ Running query...
$ cat /tmp/results.txt

Query: SELECT date, time, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-----------+--------+--------+
| date       | time     | requestip | method | status |
| 2014-08-05 | 15:56:57 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 | 10.0.0.3  | GET    |    200 |
| 2014-08-05 | 15:56:58 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:59 | 10.0.0.15 | GET    |    304 |
+------------+----------+-----------+--------+--------+
Run time: 2.41 seconds | Data scanned: 101.82 KB
Location: s3://aws-athenai-demo/62af0cf0-9417-47d4-a0c0-19250dce59a8.csv
```

or just redirect stdout to a file:

```
$ athenai run "SELECT date, time, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;" > /tmp/results.txt
⠙ Running query...
$ cat /tmp/results.txt

Query: SELECT date, time, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-----------+--------+--------+
| date       | time     | requestip | method | status |
| 2014-08-05 | 15:56:57 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 | 10.0.0.3  | GET    |    200 |
| 2014-08-05 | 15:56:58 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:59 | 10.0.0.15 | GET    |    304 |
+------------+----------+-----------+--------+--------+
Run time: 2.23 seconds | Data scanned: 101.82 KB
Location: s3://aws-athenai-demo/be70bc11-6234-4960-ab81-608749c3a4b8.csv
```


## Configuration file

You can save your configurations into `$HOME/.athenai/config` file to simplify every command execution.

### File format

Athenai's configuration file has simple INI file format.
Here are available settings:

```ini
# Section name
[default]
# Profile in your $HOME/.aws/credentials file
# Default: default
profile = default

# AWS region
# Default: us-east-1
region = us-east-1

# Database name
database = sampledb

# Output location in S3 where query results will be stored
location = s3://aws-athena-query-results-<YOUR_ACCOUNT_ID>-us-east-1/

# Encryption configuration for query results
## Encryption type
## Valid values: SSE_S3, SSE_KMS, CSE_KMS
encrypt = SSE_KMS
## KMS key ARN or ID used for SSE_KMS or CSE_KMS
kms = <YOUR_KMS_KEY_ARN_OR_ID>

# Turn on debug logging
# Default: false
debug = false

# Do not show informational messages
# Default: false
silent = false

# Output query results to a given file path instead of stdout
output = /path/to/file

# The formatting style for query results
# Valid values: table, csv
# Default: table
format = table

# The maximum possible number of SUCCEEDED query executions to list
# Default: 50
count = 50
```

**The `[default]` section is required since Athenai uses config values inside the section by default.**

You can optionally add other sections into your file. For example, add `[oregon]` section as follows:

```ini
[default]
profile = default
region = us-east-1
database = sampledb
location = s3://aws-athena-query-results-<YOUR_ACCOUNT_ID>-us-east-1/

 # Section for us-west-2
[oregon]
# Use another profile
profile = myuser
# Use us-west-2 region
region = us-west-2
# I created the database in us-west-2
database = elb_logs
# Save your query results into your other bucket
location = s3://my-elb-logs-query-results/
```

and then use `--section/-s` flag to specify the section to use:

```
$ athenai run --section oregon "SHOW DATABASES"
⠚ Running query...
Query: SHOW DATABASES;
+----------+
| default  |
| elb_logs |
| sampledb |
+----------+
Run time: 0.39 seconds | Data scanned: 0 B
Location: s3://my-elb-logs-query-results/401e35ec-6b91-4bbf-a45f-bd144b17e199.txt
```

Note that you can also specify all of the above configuration values via command line flags when running a command.
See each command's `--help` message for more details.

### Location of configuration file

By default Athenai loads `$HOME/.athenai/config` automatically and use values in the file.
If Athenai cannot find or fails to load the config file at the location, it ignores the file and uses command line flags only.

If you want to use another config file at another location, use `--config` flag to specify its path (also don't forget to specify `--section` unless `default`):

```
$ cat /tmp/myconfig
[oregon]
profile = myuser
region = us-west-2
database = elb_logs
location = s3://my-elb-logs-query-results/

$ athenai run --config /tmp/myconfig --section oregon "SHOW DATABASES"
⠳ Running query...
Query: SHOW DATABASES;
+----------+
| default  |
| elb_logs |
| sampledb |
+----------+
Run time: 0.28 seconds | Data scanned: 0 B
Location: s3://my-elb-logs-query-results/3334a6f3-2de1-4e6b-b144-32de59645cee.txt
```

### Note: precedence of configuration values

**Command line flags have higher priority than config file**, so if you specify flags explicitly when running a command, values in the config file are overridden by the flags.


## Bug report & feature request

Feel free to open an issue if you encounter any problem or have a feature request! 😄

However, in order to solve your issue quickly and avoid duplicate effort, please follow the steps below.

1. Search a similar issue already reported [here](https://github.com/skatsuta/athenai/issues?utf8=%E2%9C%93&q=is%3Aissue).
1. If it exists, add your comment or [Reaction (for +1)](https://github.com/blog/2119-add-reactions-to-pull-requests-issues-and-comments).
1. If it doesn't exist, [create a new issue](https://github.com/skatsuta/athenai/issues/new) and describe the details.


## Contributing

Your contributions are always welcome! 😆

Please follow the steps below to fix a bug or add a new feature, etc.

1. [Fork the original repo](https://github.com/skatsuta/athenai/fork)
1. **Clone the ORIGINAL repo (NOT your fork)**
   ```
   $ git clone https://github.com/skatsuta/athenai.git
   ```
1. **Add your fork as a new remote named `fork`**
   ```
   $ git remote add fork https://github.com/yourname/athenai.git
   ```
1. Create your bug fix or feature branch
   ```
   $ git checkout -b your-working-branch
   ```
1. Update the code, add tests if necessary and make sure all tests pass
   ```
   $ ./scripts/test.sh
   ```
1. Commit your changes

   Please describe the details of your commit in the commit message and include a corresponding GitHub issue number if it exists.

1. **Push to the `fork` (NOT to the `origin`)**
   ```
   $ git push fork
   ```
1. Create a new pull request against the `master` branch


## License

[Apache License 2.0](https://github.com/skatsuta/athenai/blob/master/LICENSE)


## Author

[Soshi Katsuta (skatsuta)](https://github.com/skatsuta)


