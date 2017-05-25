# Athenai

Athenai is a simple command line tool that accesses Amazon Athena.


## Description

Athenai is a simple command line tool that accesses Amazon Athena.

TODO

## Demo

![Demo](https://image-url.gif)


## Features

- Easy execution: Run a query from stdin or an SQL file and output results after the execution has finished.
- Concurrent executions support: run multiple queries in one command.
- Several output formats: Either table, CSV, or raw JSON.
- Query cancellation: Cancel a query if Ctrl-C is pressed while the query is running.
- Named queries: Manage and run named queries easily.


## Requirement

You need to set up AWS credentials before using this tool.

Additionally, you can optionally save the default option values into `~/.athenairc` to make command line options simple.

```toml
[default]
database = sampledb
output = s3://aws-athena-query-results-123456789012-us-east-1/
```

See config file section for more details.


## Usage

In this section I omit config options to describe main usage simply.
If you want to specify config options explicitly or override default options in `.athenairc`, run commands with options like this:

```bash
$ athenai --database sampledb --output s3://aws-athena-query-results-123456789012-us-east-1/ "SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;"
```

### Running a single query

To run a single query, pass it as an argument:

```bash
$ athenai run "SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;"
Running query...
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 kB
```

### Running queries interactively

To run queries interactively, run a command with no argument:

```bash
$ athenai run
> SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;
Running query...
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 kB
> SHOW DATABASES;
Running Query..
+-----------------+
| cloudfront_logs |
| default         |
| elb_logs        |
| s3_logs         |
| sampledb        |
+-----------------+
Run time: 0.322 seconds | Data scanned: 0 B
```

### Running queries from an SQL file

To run a query from an SQL file, pass its file path with `file://` prefix:

```bash
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;
$ athenai run file://sample.sql
Running query...
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 kB
```

or pass its content via stdin if you can use pipe on Unix-like OS:

```bash
$ cat sample.sql
SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs LIMIT 5;
$ athenai < sample.sql
Running query...
+------------+----------+-------+-----------+--------+--------+
| date       | time     | bytes | requestip | method | status |
| 2014-07-05 | 15:00:00 |  4260 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |    10 | 10.0.0.15 | GET    |    304 |
| 2014-07-05 | 15:00:00 |  4252 | 10.0.0.15 | GET    |    200 |
| 2014-07-05 | 15:00:00 |  4257 | 10.0.0.8  | GET    |    200 |
| 2014-07-05 | 15:00:03 |  4261 | 10.0.0.15 | GET    |    200 |
+------------+----------+-------+-----------+--------+--------+
Run time: 2.149 seconds | Data scanned: 101 kB
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
> Query: SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs ORDER BY date, time DESC LIMIT 5;
```

or create a named query in one liner:

```bash
$ athenai named create --database sampledb --name "Show the latest 5 records" "SELECT date, time, bytes, requestip, method, status FROM cloudfront_logs ORDER BY date, time DESC LIMIT 5;"
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
$ wget https://.../athenai.zip
$ unzip athenai.zip
$ mv athenai ~/bin/
```

Alternatively, you can use `go get` if you have installed Go:

```bash
$ go get -u -v github.com/skatsuta/athenai
$ athenai --version
```


## Licence

[MIT](https://github.com/skatsuta/athenai/blob/master/LICENCE)

## Author

[skatsuta](https://github.com/skatsuta)


