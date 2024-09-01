shui
====

[![Go Report Card](https://goreportcard.com/badge/github.com/spacez320/shui)](https://goreportcard.com/report/github.com/spacez320/shui)
[![Go Reference](https://pkg.go.dev/badge/github.com/spacez320/shui/cmd/shui.svg)](https://pkg.go.dev/github.com/spacez320/shui/cmd/shui)
![GitHub Release](https://img.shields.io/github/v/release/spacez320/shui)

Shui is a tool that can be used to extract information with a CLI and observe it over time. It is
meant to aid system administrators, platform engineers, or anyone who spends time doing operations
in a command line.

- It's like a better `watch` that can draw graphs, tables, etc.
- It's like a simplified metrics database that can run directly in your console.
- It can act as a bridge between a command line and external monitoring systems.

This project is in an active, "alpha" development phase, but should generally be useable.

Setup
-----

- Binaries are available from [the releases page](https://github.com/spacez320/shui/releases).
- Docker images are available on [Docker Hub](https://hub.docker.com/repository/docker/spacez320/shui).

You can also just:

```sh
go install github.com/spacez320/shui/cmd/shui@latest
```

Usage
-----

Shui expects a **query** to gather data with (e.g. a CLI command, a process ID, etc.). Queries
produce **results** that are parsed automatically and stored as a time series.

> Note: I'm not good at making gifs, so some of the commands shown may be outdated, even if the
> functionality isn't.

### Modes

Shui has **"modes"** that determine what type of query should be provided.

**Query mode** is the default and is for running shell commands.

![Demo of query mode](https://raw.githubusercontent.com/spacez320/shui/master/assets/query-mode.gif)

**Profile mode** is like Query mode except specialized for inspecting systems or processes.

![Demo of profile mode](https://raw.githubusercontent.com/spacez320/shui/master/assets/profile-mode.gif)

### Displays

Shui also has **"displays"** that determine how data is presented.

**Raw display** and **stream display** simply presents incoming data, the latter being within Shui's
interactive window. The examples above use stream displays.

**Table display** will parse results into a table.

![Demo of table display](https://raw.githubusercontent.com/spacez320/shui/master/assets/table-display.gif)

**Graph display** will target a specific field in a result and graph it (this requires the query to
produce a number).

![Demo of graph display](https://raw.githubusercontent.com/spacez320/shui/master/assets/graph-display.gif)

### Examples

These examples show basic usage.

> The examples below have been tested on `GNU bash, version 5.2.15(1)-release`.

```sh
# See help.
shui -h

# Execute `whoami` once, printing results to the console and waiting for a user to `^C`.
shui -query 'whoami'

# Execute `uptime` continuously, printing results to the console, without using persistence.
shui \
    -count -1 \
    -query 'uptime' \
    -store=none

# Get the size of an NVME disk's used space and output it to a table with the specific label "NVME
# Used Space".
shui \
    -count -1 \
    -display 3 \
    -labels "NVME Used Space" \
    -query 'df -h | grep nvme0n1p2 | awk '\''{print $3}'\'''
```

### Integrations

Shui can send its data off to external systems, making it useful as an ad-hoc metrics or log
exporter. Supported integrations are listed below.

#### Elasticsearch

Shui can create Elasticsearch documents from results.

```sh
shui \
    -elasticsearch-addr <addr> \
    -elasticsearch-index <index> \
    -elasticsearch-user <user> \
    -elasticsearch-password <password
```

- Documents are structured according to result labels supplied with `-labels`, prefixed with
  `shui.value.`.
- Documents will also contain an additional field, `shui.query`.
- The result `Time` field will be mapped to `timestamp`.
- Shui must use HTTP Basic Auth (credentials are given with `-elasticsearch-user` and
  `-elasticsearch-password`).
- Shui will not attempt to create an index (one must be supplied with `-elasticsearch-index`).

As an example, given a query `cat file.txt | wc` and `-labels "newline,words,bytes"`, the following
Elasticsearch document would be created:

```json
{
    "_index": "some-index",
    "_id": "some-id",
    "_score": 1.0,
    "_source": {
        "shui.query": "cat file.txt | wc",
        "shui.value.bytes": 3,
        "shui.value.newline": 1,
        "shui.value.words": 2,
        "timestamp": "2024-06-10T17:40:29.773550719-04:00"
    }
}
```

#### Prometheus

Shui can create Prometheus metrics from numerical results. Both normal Prometheus collection
and Pushgateway are supported.

```sh
# Start a Prometheus collection HTTP page.
shui -prometheus-exporter <address>

# Specify a Prometheus Pushgateway address to send results to.
shui -prometheus-pushgateway <address>
```

- Metrics namse will have the structure `shui_<query>` where `<query>` will be changed to
  conform to Prometheus naming rules.
- Shui labels supplied with `-labels` will be saved as a Prometheus label called
  `shui_label`, creating a unique series for each value in a series of results.

As an example, given a query `cat file.txt | wc`, and `-labels "newline,words,bytes"`, the following
Prometheus metrics would be created:

```
shui_cat_file_txt_wc{shui_label="newline"}
shui_cat_file_txt_wc{shui_label="words"}
shui_cat_file_txt_wc{shui_label="bytes"}
```

> **NOTE:** The only currently supported metric is a **Gauge** and queries must provide something
> numerical to be recorded.

### Persistence

Shui, by default, will store results and load them when re-executing the same query.

The only currently supported storage is local disk, located in the user's cache directory. See:
<https://pkg.go.dev/os#UserCacheDir>.

### Expressions

Shui has the ability to execute "expressions" on query results in order to manipulate them
before display (e.g. performing statistics, combining values, producing cumulative series, etc.).

Some key points about expressions:

- Multiple expressions may be provided and execute in the order provided.
- Filters apply before expressions.
- It uses Expr, a Go-centric expression language.
- The expression language is type sensitive, but results of expressions will always be strings.

Expressions are able to access variables:

1.  `result`, a map of the current result's labels to values.
2.  `prevResult`, the previous result mapping, for cumulative results. Note that expressions must
    account for `prevResult` being an empty map for the first result in a series.

Some examples:

```sh
# Multiply the 5m CPU average by 10. Note that we invoke `get` with a key of `"9"` because default
# labels are string indexes and no labels were provided.
shui -query 'uptime | tr -d ","' -expr 'get(result, "9") * 10'

# Cumulatively sum 5m CPU average. Note that we need to account for prevResult being empty and we
# must convert the prevResult from a string to a float.
shui -query 'uptime | tr -d ","' -filters 9 -expr 'get(result, "0") + ("0" in prevResult?
float(get(prevResult, "0")) : 0)'
```

See: <https://expr-lang.org/docs/language-definition>

Future
------

I've been adding planned work into [project issues](https://github.com/spacez320/shui/issues)
and [project milestones](https://github.com/spacez320/shui/milestone/1)--take a look there to
see what's coming or to make suggestions.

Planned improvements include things like:

- [ ] Background execution.
- [x] Persistent results.
- [x] Ability to perform calculations on streams of data, such as aggregates, rates, or quantile math.
- [ ] Better text result management, such as diff'ing.
- [x] Export data to external systems, such as Prometheus.
- [x] ... and Elasticsearch.
- [ ] More detailed and varied display modes.
- [ ] Historical querying.
- [ ] Beter management of textual data, including diffs.

Similar Projects
----------------

There doesn't seem to be much out there easily visible that matches the same set of functionality,
but there are a few projects I've found that do some things similarly.

- [DataDash](https://github.com/keithknott26/datadash), a data visualization tool for the terminal.
- [Grafterm](https://github.com/slok/grafterm), visualize metrics dashboards on the terminal.
- [Euoporie](https://github.com/joouha/euporie), a terminal interactive computing environment for
  Jupyter.
