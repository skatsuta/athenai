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


## Requirement

You need to set up AWS credentials before using this tool.

TODO: setup document links

You can optionally save your default option values into `~/.athenai/config` to simplify every command execution.

```toml
[default]
region = us-east-1
database = sampledb
output = s3://aws-athena-query-results-123456789012-us-east-1/
```

**Note**: Athenai does not read default settings in `~/.aws/config`, so you need to set the `region` option explicitly.

See config file section for more details.

## Usage

#### Note: config option flags

In this section I omit config option flags to describe the main usage simply.
If you want to specify the options explicitly or override default options in `.athenai/config.yml`, run a command like this:

```bash
$ athenai run --region us-east-1 --database sampledb --output s3://aws-athena-query-results-123456789012-us-east-1/ "SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;"
```

#### Note: the order of query results

The order of query results may be different from that of given queries, because by default Athenai makes all query execution requests concurrently to Amazon Athena, and shows the results in the order completed.

You can use `--order` option if you want to display the results in the same order.

### Running queries interactively (REPL mode)

To run queries on interactive (REPL) mode, run `athenai run` command with no arguments:

```bash
$ athenai run
athenai> SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;
Running query...
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
Running query...
SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.38 seconds | Data scanned: 0 B

SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.35 seconds | Data scanned: 0 B
athenai> 
```

Press `Ctrl-C` or `Ctrl-D` on empty line to exit.

### Running queries from command line arguments

To run queries from command line arguments, just pass them to `athenai run` command:

```bash
$ athenai run "SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;"
Running query...
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
Running query..
SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 0.40 seconds | Data scanned: 0 B
.
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.34 seconds | Data scanned: 0 B
```


### Running queries from an SQL file

To run queries from an SQL file, pass its file path with `file://` prefix to `athenai run` command:

```bash
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;

$ athenai run file://sample.sql
Running query...
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

or pass its content via stdin if you can use pipe on Unix-like OS:

```bash
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM sampledb.cloudfront_logs LIMIT 5;

$ athenai run < sample.sql
Running query...
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


## Installation

Simply download the binary and place it in `$PATH`:

```bash
$ curl -O https://.../athenai.zip
$ unzip athenai.zip
$ mv athenai /usr/local/bin/ # or where you like
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


