<p align="center"><img src="docs/logo.jpg" alt="logo"></p>


# Athenai

[![wercker status](https://app.wercker.com/status/554177081896c1c508e365ffa1c8cc26/s/master "wercker status")](https://app.wercker.com/project/byKey/554177081896c1c508e365ffa1c8cc26)
[![codecov](https://codecov.io/gh/skatsuta/athenai/branch/master/graph/badge.svg?token=dkRnWrYYa9)](https://codecov.io/gh/skatsuta/athenai)

Have fun with Amazon Athena from command line! 🕊


## Overview

Athenai is a simple and easy-to-use command line tool that runs SQL statements on [Amazon Athena](https://aws.amazon.com/athena/).

With Athenai you can easily run multiple queries at a time on Amazon Athena and see the results in table or CSV format once the executions are complete.

"A picture is worth a thousand words." See the **[Demo](#demo)** section to see how it works 👀


## Demo

![Demo](docs/demo.gif)


## Features

- **Easy to use**: give queries, wait for query executions and see the results once the executions have finished.
- **Various input methods**: REPL, command line arguments or an SQL file.
- **Concurrency support**: run multiple queries concurrently in one command.
- **Query cancellation**: cancel queries if Ctrl-C is pressed while the queries are running.


## Installation

Simply download the binary and place it in `$PATH`:

```bash
$ curl -O https://.../athenai.zip
$ unzip athenai.zip
$ mv athenai /usr/local/bin/ # or wherever you like
$ athenai --help
```


## Setup

### AWS creadentitals (Required)

Before using this tool, you need to set up AWS credentials just like using AWS CLI or AWS SDK.
If you already use AWS CLI or AWS SDK, you probably do not need this step and just skip to the next **[Default configuration file](#default-configuration-file-optional-)** section! 🚀

To set up AWS credentials, there are mainly three ways:

* [Configuring environment variables](http://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html): simple, but not recommended.
* [Configuring shared credentials file (`~/.aws/credentials`)](http://docs.aws.amazon.com/cli/latest/userguide/cli-config-files.html): recommended for use on local computers.
* [Configuring instance profile](http://docs.aws.amazon.com/cli/latest/userguide/cli-metadata.html): recommended for use on EC2 instances.

Please follow one of the above instructions corresponding to your use case.

After set it up, check Athena and S3 permissions in your IAM user or role policy to use.
**To use Amazon Athena, `athena:*` and `s3:*` permissions are required to be allowed in your IAM policy.**

### Default configuration file (Optional)

You can optionally set your default configuration values into `~/.athenai/config` to simplify every command execution.

Write the following into `~/.athenai/config` and save it.

```ini
[default]
profile = default
region = us-east-1
database = sampledb
location = s3://aws-athena-query-results-[YOUR_ACCOUNT_ID]-us-east-1/
```

Afterwards Athenai loads the configuration automatically and you can omit the option flags when running commands.

See the **[Configuration file](#configuration-file)** section for more details.


## Usage

#### Note: config option flags

In this section I omit config option flags to describe the main usage simply.
If you haven't set up the `~/.athenai/config` file yet or want to override default options in the config file, run a command with flags as follows:

```
$ athenai run \
  --profile default \
  --region us-east-1 \
  --database sampledb \
  --location s3://aws-athena-query-results-[YOUR_ACCOUNT_ID]-us-east-1/ \
  "SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;"
```

### Running queries interactively (REPL mode)

![Running queries interactively](docs/run_repl.gif)

To run queries on interactive (REPL) mode, run `athenai run` command with no arguments except flags:

```
$ athenai run
athenai> SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
⠞ Running query...
QueryExecutionId: c4a4b47a-5ec6-4cc1-8e52-6a8c14b28f7a
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.27 seconds | Data scanned: 101.27 KB
athenai> SHOW DATABASES; SHOW TABLES;
⠴ Running query...
QueryExecutionId: e0db931c-f69a-4a3e-9fe5-d5baf9e80811
Query: SHOW DATABASES;
+-----------------+
| cloud_trail     |
| cloudfront_logs |
| default         |
| elb_logs        |
| s3_logs         |
| sampledb        |
| test            |
+-----------------+
Run time: 0.34 seconds | Data scanned: 0 B

QueryExecutionId: c1a8f451-b172-4dd6-a35b-378a49e6856e
Query: SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.87 seconds | Data scanned: 0 B
athenai> ^D
$ 
```

In REPL mode you can use common key shortcuts just like on most shells. For example,

Key | Action
:---:|---
`↑`/`Ctrl-P` | Go to the previous history
`↓`/`Ctrl-N` | Go to the next history
`Ctrl-A` | Go to the beginning of the line
`Ctrl-E` | Go to the end of the line
`Ctrl-H` | Delete a character

... and so on.
Available shortcuts are listed [here](https://github.com/chzyer/readline/blob/master/doc/shortcut.md).

Query history is saved to `~/.athenai/history` automatically.

To exit REPL, press `Ctrl-C` or `Ctrl-D` on empty line.

### Running queries from command line arguments

![Running queries from command line arguments](docs/run_arg.gif)

To run queries from command line arguments, just pass them to `athenai run` command:

```
$ athenai run "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠴ Running query...
QueryExecutionId: dbce2e27-90f8-480d-99d7-1ea1af3be962
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.18 seconds | Data scanned: 101.27 KB

$ athenai run "SHOW DATABASES; SHOW TABLES;"
⠚ Running query...
QueryExecutionId: ab6a1804-1827-4b82-9ba2-f50c61488df7
Query: SHOW DATABASES;
+-----------------+
| cloud_trail     |
| cloudfront_logs |
| default         |
| elb_logs        |
| s3_logs         |
| sampledb        |
| test            |
+-----------------+
Run time: 0.31 seconds | Data scanned: 0 B

QueryExecutionId: cbbf3e73-7653-4788-9cc3-57df5ac5196a
Query: SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.37 seconds | Data scanned: 0 B
```

### Running queries from an SQL file

![Running queries from an SQL file](docs/run_file.gif)

To run queries from an SQL file, pass its file path with `file://` prefix to `athenai run` command:

```
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;

$ athenai run file://sample.sql
⠲ Running query...
QueryExecutionId: 69ba40e7-9623-4e57-b5ac-8d05ce3c50c7
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.08 seconds | Data scanned: 101.27 KB
```

or pass its content via stdin:

```
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;

$ athenai run < sample.sql
⠚ Running query...
QueryExecutionId: 7226c8a5-c3b6-4399-97fb-ea7683e774d1
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 1.90 seconds | Data scanned: 101.27 KB
```

### Canceling query executions

![Canceling query executions](docs/run_cancel.gif)

You can cancel query executions by pressing `Ctrl-C` while they are running.

```
$ athenai run "SELECT * FROM sampledb.cloudfront_logs"   # Oops! Full scan by mistake!
⠖ Running query... ^C
⠋ Canceling...
$ # Whew! That was close.
```

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

Athenai uses [peco/peco](https://github.com/peco/peco) as an interactive filtering library.
Frequently used key mappings are:

Key | Action
:---:|---
`↑`/`Ctrl-P` | Move up
`↓`/`Ctrl-N` | Move down
`Ctrl-Space` | Select/unselect each entry
`Ctrl-A` | Go to the beginning of the QUERY line
`Ctrl-E` | Go to the end of the QUERY line
`Ctrl-H` | Delete a character in the QUERY line

Available key mappings are listed [here](https://github.com/peco/peco#default-keymap).
By default you can select multiple entries by pressing `Ctrl-Space` on each entry.


After you have selected the entries to show, hit `Enter` and you will see the results of selected query executions like the following:

```
$ athenai show
⠦ Loading history...
⠚ Fetching results...
QueryExecutionId: c85750eb-4f7a-485c-9a50-bd648b69b617
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-08-05 | 15:56:57 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4257 | 10.0.0.3  | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:58 |  4251 | 10.0.0.15 | GET    |    200 |
| 2014-08-05 | 15:56:59 |    10 | 10.0.0.15 | GET    |    304 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.06 seconds | Data scanned: 203.09 KB

QueryExecutionId: 91f87c1c-11e5-4458-911a-ef0c059c0c71
Query: SHOW DATABASES;
+-----------------+
| cloud_trail     |
| cloudfront_logs |
| default         |
| elb_logs        |
| s3_logs         |
| sampledb        |
| test            |
+-----------------+
Run time: 0.28 seconds | Data scanned: 0 B
```

By default the `show` command lists up to the latest 50 query executions except ones in `FAILED` state.
You can configure the number by specifying `--count/-c` flag:

```
$ athenai show --count 100
```

If you want to list all of your completed query executions so far, specify `0`:

```
$ athenai show --count 0
```

Note that `athenai show --count 0` may be very slow depending on the total number of your query executions.

### Printing results in CSV format

![Printing results in CSV format](docs/format_csv.gif)

If you want to print query results in CSV format, specify `--format/-f csv` flag.

```
$ athenai run --format csv "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠙ Running query...
QueryExecutionId: 70941c18-d22a-4a90-ad93-09a8b22659e8
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
date,time,bytes,requestip,method,status
2014-07-05,15:00:00,4260,10.0.0.15,GET,200
2014-07-05,15:00:00,10,10.0.0.15,GET,304
2014-07-05,15:00:00,4252,10.0.0.15,GET,200
2014-07-05,15:00:00,4257,10.0.0.8,GET,200
2014-07-05,15:00:03,4261,10.0.0.15,GET,200
Run time: 2.06 seconds | Data scanned: 101.27 KB
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

### Outputting (Saving) results to a file

![Outputting (Saving) results to a file](docs/run_output.gif)

If you want to output (save) the query results to a file, use `--output/-o` flag to specify the file path:

```
$ athenai run --output /tmp/results.txt "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠴ Running query...
$ cat /tmp/results.txt

QueryExecutionId: c4ba05d0-3931-4aca-aa45-d937cbc009e1
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.29 seconds | Data scanned: 101.27 KB
```

or just redirect stdout to a file:

```
$ athenai run "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;" > /tmp/results.txt
⠋ Running query...
$ cat /tmp/results.txt

QueryExecutionId: 0e17b02c-94a8-45c0-810b-1765073a7561
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.45 seconds | Data scanned: 101.27 KB
```


## Configuration file

You can save your configurations into `~/.athenai/config` file to simplify every command execution.

### File format

Athenai's configuration file has simple INI file format like this:

```ini
[default]  # Section
profile = default  # Profile in your ~/.aws/credentials file
region = us-east-1  # AWS region to use
database = sampledb  # Database name to use
location = s3://aws-athena-query-results-123456789012-us-east-1/  # Output location in S3 where query results will be stored
```

**The `[default]` section is required since Athenai uses config values inside the section by default.**

You can optionally add the arbitrary number of other sections into your file. For example,

```ini
[default]
profile = default
region = us-east-1
database = sampledb
location = s3://aws-athena-query-results-[YOUR_ACCOUNT_ID]-us-east-1/

[cf_logs]  # Section for cloudfront_logs
profile = myuser  # Use another profile
region = us-west-2  # Use us-west-2 region
database = cloudfront_logs  # I created the database in us-west-2
location = s3://my-cloudfront-logs-query-results/  # Save your query results into your other bucket
```

Then use `--section/-s` flag to specify the section to use:

```
$ athenai run --section cf_logs "SHOW DATABASES"
⠳ Running query...
QueryExecutionId: 44433dcf-d90b-4230-90a0-deb458a26624
Query: SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| sampledb        |
+-----------------+
Run time: 0.34 seconds | Data scanned: 0 B
```

### Location of configuration file

By default Athenai loads `~/.athenai/config` automatically and use values in the file.
If Athenai cannot find the config file in the location or fails to load the file, it ignores the file and uses command line flags only.

If you want to use another config file in another location, use `--config` flag to specify its path (also don't forget to specify `--section` unless `default`):

```
$ cat /tmp/myconfig
[cf_logs]
profile = myuser
region = us-west-2
database = cloudfront_logs
location = s3://my-cloudfront-logs-query-results/

$ athenai run --config /tmp/myconfig --section cf_logs "SHOW DATABASES"
⠳ Running query...
QueryExecutionId: 3334a6f3-2de1-4e6b-b144-32de59645cee
Query: SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| sampledb        |
+-----------------+
Run time: 0.31 seconds | Data scanned: 0 B
```

### Note: precedence of configuration values

Option flags have higher priority than config file, so if you specify option flags explicitly when running a command, values in the config file are overridden by the flags.


## Bug report

[Create a new GitHub issue](https://github.com/skatsuta/athenai/issues/new).


## Contributing

Please follow the steps below to fix a bug or add a new feature, etc.

1. [Fork the original repo](https://github.com/skatsuta/athenai/fork)
1. **Clone the original repo (NOT your forked one)**
  * `git clone https://github.com/skatsuta/athenai.git`
1. **Add your forked repo as a new `fork` remote**
  * `git remote add fork https://github.com/yourname/athenai.git`
1. Create your bug fix or feature branch
  * e.g. `git checkout -b your-working-branch`
1. Change the code, add tests if necessary and make sure all tests passes
  * `./scripts/test.sh`
1. Commit your changes (please describe details of your commit in the commit message)
  * e.g. `git commit -am 'Fix a bug'`
1. **Push to the `fork` branch (NOT to the `origin`)**
  * `git push fork`
1. Create a new Pull Request against the `master` branch


## License

[Apache License 2.0](https://github.com/skatsuta/athenai/blob/master/LICENCE)


## Author

[Soshi Katsuta (skatsuta)](https://github.com/skatsuta)


