# Athenai

Athenai is a simple command line tool that accesses Amazon Athena.


## Description

Athenai is a simple command line tool that accesses Amazon Athena.

TODO

## Demo

![Demo](https://image-url.gif)


## Features

- Easy to use: give queries, wait for query executions and see the results after the executions have finished.
- Various input methods: REPL, command line arguments or an SQL file.
- Concurrency support: run multiple queries concurrently in one command.
- Several output formats: table, CSV, JSON, or raw file on S3.
- Query cancellation: Cancel queries if Ctrl-C is pressed while the queries are running.
- Named queries: Manage and run named queries easily.


## Setup

### AWS Creadentitals (Required)

You need to set up AWS credentials before using this tool.

TODO: setup document links

### Default configuration file (Optional)

You can optionally set your default configuration values into `~/.athenai/config` to simplify every command execution.

Write the following into `~/.athenai/config` and save it.

```toml
[default]
profile = default
region = us-east-1
database = sampledb
location = s3://aws-athena-query-results-[YOUR_ACCOUNT_ID]-us-east-1/
```

Afterwards Athenai loads the configuration automatically and you can omit the option flags when running commands.

See the **Configuration file** section for more details.

## Getting Started

#### Note: config option flags

In this section I omit config option flags to describe the main usage simply.
If you haven't set up the `~/.athenai/config` file yet or want to override default options in the config file, run a command with options like this:

```bash
$ athenai run \
  --profile default \
  --region us-east-1 \
  --database sampledb \
  --location s3://aws-athena-query-results-[YOUR_ACCOUNT_ID]-us-east-1/ \
  "SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;"
```

### Running queries interactively (REPL mode)

To run queries on interactive (REPL) mode, run `athenai run` command with no arguments except flags:

```bash
$ athenai run
athenai> SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
⠳ Running query...
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 KB
athenai> SHOW DATABASES; SHOW TABLES;
⠳ Running query...
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.35 seconds | Data scanned: 0 B

SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.38 seconds | Data scanned: 0 B
athenai> 
```

Press `Ctrl-C` or `Ctrl-D` on empty line to exit.

### Running queries from command line arguments

To run queries from command line arguments, just pass them to `athenai run` command:

```bash
$ athenai run "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠳ Running query...
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 KB

$ athenai run "SHOW DATABASES; SHOW TABLES;"
⠳ Running query...
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.34 seconds | Data scanned: 0 B

SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.40 seconds | Data scanned: 0 B
```

If you want to display query results in CSV format, specify `--format/-f csv` flag.

```bash
$ time go run main.go run --format csv "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
⠚ Running query...
QueryExecutionId: 922019ee-d2d7-4b45-bf86-696ca5cc343a
Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
date,time,bytes,requestip,method,status
2014-07-05,15:00:00,4260,10.0.0.15,GET,200
2014-07-05,15:00:00,10,10.0.0.15,GET,304
2014-07-05,15:00:00,4252,10.0.0.15,GET,200
2014-07-05,15:00:00,4257,10.0.0.8,GET,200
2014-07-05,15:00:03,4261,10.0.0.15,GET,200
Run time: 3.18 seconds | Data scanned: 101.27 KB
```

You can also use this flag with `athenai show` command described later.

### Running queries from an SQL file

To run queries from an SQL file, pass its file path with `file://` prefix to `athenai run` command:

```bash
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;

$ athenai run file://sample.sql
⠳ Running query...
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 KB
```

or pass its content via STDIN:

```bash
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;

$ athenai run < sample.sql
⠳ Running query...
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 KB
```

### Showing results of completed query executions

Run the below command:

```bash
$ athenai show
```

and you can select query executions you want to show, interactively with `Ctrl-Space`:

```bash
QUERY>                                                                                                                                                IgnoreCase [48 (1/1)]
2017-06-26 14:11:36 +0000 UTC   SHOW TABLES SUCCEEDED   0.37 seconds    0 B
2017-06-26 14:11:36 +0000 UTC   SELECT timestamp, requestip, backendip FROM elb_logs LIMIT 3   SUCCEEDED   0.55 seconds    17.80 KB
2017-06-26 14:11:36 +0000 UTC   SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 10   SUCCEEDED   2.23 seconds    101.27 KB
2017-06-26 14:11:36 +0000 UTC   SHOW DATABASES  SUCCEEDED   0.38 seconds    0 B
(snip)
```

Then hit `Enter` and you will see the results of selected query executions like the following:

```bash
⠦ Loading history...
⠋ Fetching results...
QueryExecutionId: 22917c12-218d-463c-b9e7-ec7f03aa3588
Query: SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| default         |
| elb_logs        |
| sampledb        |
+-----------------+
Run time: 0.38 seconds | Data scanned: 0 B

QueryExecutionId: bb2b0930-9e3e-47fb-9985-17c0ff2cbd89
Query: SELECT timestamp, requestip, backendip FROM elb_logs LIMIT 3;
+-----------------------------+----------------+-----------------+
| timestamp                   | requestip      | backendip       |
| 2014-09-27T00:00:25.424956Z | 241.230.198.83 | 251.192.40.76   |
| 2014-09-27T00:00:56.439218Z | 252.26.60.51   | 249.89.116.3    |
| 2014-09-27T00:01:27.441734Z | 250.244.20.109 | 251.111.156.171 |
+-----------------------------+----------------+-----------------+
Run time: 0.55 seconds | Data scanned: 17.80 KB
```

By default the `show` command lists up to 50 query executions except those in FAILED state, and you can configure the number by using `--count/-c` flag:

```bash
$ athenai show --count 100
```

If you want to list all of your completed query executions, specify `0`:

```bash
$ athenai show --count 0
```

Note that `--count 0` may be very slow depending on the total number of your query executions.

### Manage and run named queries

#### List named queries

```bash
$ athenai named list
```

#### Create a named query

Create a named query interactively:

```bash
$ athenai named create
> Database: sampledb
> Name: Show the latest 5 records
> Query: SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs ORDER BY date, time DESC LIMIT 5;
```

or create a named query in one liner:

```bash
$ athenai named create --database sampledb --name "Show the latest 5 records" "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs ORDER BY date, time DESC LIMIT 5;"
```


#### Delete a named query

```bash
$ athenai named delete
```

#### Run a named query

```bash
$ athenai named run
```


## Configuration file

You can save your configurations into `~/.athenai/config` file to simplify every command execution.

### File format

Athenai's configuration file has simple INI file format like this:

```ini
[default]  # Section
profile = default  # Profile in your ~/.aws/credentials file
region = us-east-1  # AWS region to use
database = sampledb  # Database name in Athena
location = s3://aws-athena-query-results-123456789012-us-east-1/  # Output location in S3 where query results are stored
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
location = s3://my-cloudfront-logs-query-results/  # Save your query results into your favorite bucket
```

Then use `--section/-s` flag to specify the section to use:

```bash
$ athenai run --section cf_logs "SHOW DATABASES"
⠳ Running query...
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| sampledb        |
+-----------------+
Run time: 0.34 seconds | Data scanned: 0 B
```

### Location of configuration file

By default Athenai automatically loads `~/.athenai/config` and use values in the file.
If Athenai cannot find the config file in the location or fails to load the file, it ignores the file and uses command line flags only.

If you want to use another config file in another location, use `--config` flag to specify its path (also don't forget to specify `--section` unless `default`):

```bash
$ cat /tmp/myconfig
[cf_logs]
profile = myuser
region = us-west-2
database = cloudfront_logs
location = s3://my-cloudfront-logs-query-results/

$ athenai run --config /tmp/myconfig --section cf_logs "SHOW DATABASES"
⠳ Running query...
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| sampledb        |
+-----------------+
Run time: 0.31 seconds | Data scanned: 0 B
```

### Note

Option flags have higher priority than config file, so if you specify option flags explicitly when running a command, values in the config file are overridden by the flags.


## Installation

Simply download the binary and place it in `$PATH`:

```bash
$ curl -O https://.../athenai.zip
$ unzip athenai.zip
$ mv athenai /usr/local/bin/ # or wherever you like
$ athenai --help
```

Alternatively, you can use `go get` if you have installed Go:

```bash
$ go get -u -v github.com/skatsuta/athenai
$ athenai --help
```


## Licence

[MIT](https://github.com/skatsuta/athenai/blob/master/LICENCE)

## Author

[skatsuta](https://github.com/skatsuta)


