cryptarch
=========

Cryptarch is a tool that can be used to extract information with a CLI and observe it over time. It
is meant to aid system administrators, platform engineers, or anyone who spends time doing
operations in a command line.

- It's like a better `watch` that can draw graphs, tables, etc.
- It's like a simplified Prometheus that can run directly in your console.

This project is in an active, "alpha" development phase, but should generally be useable.

Setup
-----

- Binaries are available from [the releases page](https://github.com/spacez320/cryptarch/releases).
- Docker images are available on [Docker Hub](https://hub.docker.com/repository/docker/spacez320/cryptarch).

Usage
-----

Cryptarch expects a **query** to gather data with (e.g. a CLI command, a process ID, etc.). Queries
produce **results** that are parsed automatically and stored as a time series.

### Modes

Cryptarch has **"modes"** that determine what type of query should be provided.

**Query mode** is the default and is for running shell commands.

![Demo of query mode](https://raw.githubusercontent.com/spacez320/cryptarch/master/assets/query-mode.gif)

**Profile mode** is like Query mode except specialized for inspecting systems or processes.

![Demo of profile mode](https://raw.githubusercontent.com/spacez320/cryptarch/master/assets/profile-mode.gif)

### Displays

Cryptarch also has **"displays"** that determine how data is presented.

**Raw display** and **stream display** simply presents incoming data, the latter being within
Cryptarch's interactive window. The examples above use stream displays.

**Table display** will parse results into a table.

![Demo of table display](https://raw.githubusercontent.com/spacez320/cryptarch/master/assets/table-display.gif)

**Graph display** will target a specific field in a result and graph it (this requires the query to
produce a number).

![Demo of graph display](https://raw.githubusercontent.com/spacez320/cryptarch/master/assets/graph-display.gif)

### Persistence

Cryptarch, by default, will store results and load them when re-executing the same query.

The only currently supported storage is local disk, located in the user's cache directory.

See: <https://pkg.go.dev/os#UserCacheDir>

### Integrations

Cryptarch can send its data off to external systems, making it useful as an ad-hoc metrics or log
exporter. Supported integrations are listed below.

#### Prometheus

Both normal Prometheus collection and Pushgateway are supported.

```sh
# Start a Prometheus collection HTTP page.
cryptarch --prometheus <address>

# Specify a Prometheus Pushgateway address to send results to.
cryptarch --pushgateway <address>
```

- Metrics name will have the structure `cryptarch_<query>` where `<query>` will be changed to
  conform to Prometheus naming rules.
- Cryptarch labels supplied with the `-labels` will be saved as a Prometheus label called
  `cryptarch_label`.

As an example, given a query `cat file.txt | wc`, and `-labels "newline,words,bytes"`, the following
Prometheus metrics would be created:

```
cryptarch_cat_file_txt_wc{cryptarch_label="newline"}
cryptarch_cat_file_txt_wc{cryptarch_label="words"}
cryptarch_cat_file_txt_wc{cryptarch_label="bytes"}
```

> **NOTE:** The only currently supported metric is a **Gauge** and queries must provide something
> numerical to be recorded.

### More Examples

> The examples below have been tested on `GNU bash, version 5.2.15(1)-release`.

```sh
# See help.
cryptarch -h

# Execute `whoami` once, printing results to the console and waiting for a user to `^C`.
cryptarch -query 'whoami'

# Execute `uptime` continuously, printing results to the console, without using persistence.
cryptarch \
    -count -1 \
    -query 'uptime' \
    -store=none

# Get the size of an NVME disk's used space and output it to a table with the specific label "NVME
Used Space".
cryptarch \
    -count -1 \
    -display 3 \
    -labels "NVME Used Space" \
    -query 'df -h | grep nvme0n1p2 | awk '\''{print $3}'\'''
```

Future
------

I've been adding planned work into [project issues](https://github.com/spacez320/cryptarch/issues)
and [project milestones](https://github.com/spacez320/cryptarch/milestone/1)--take a look there to
see what's coming or to make suggestions.

Planned improvements include things like:

- [ ] Background execution.
- [x] Persistent results.
- [ ] Ability to perform calculations on streams of data, such as aggregates, rates, or quantile math.
- [ ] Better text result management, such as diff'ing.
- [x] Export data to external systems, such as Prometheus.
- [ ] ... and Elasticsearch.
- [ ] More detailed and varied display modes.
- [ ] Historical querying.

Similar Projects
----------------

There doesn't seem to be much out there easily visible that matches the same set of functionality,
but there are a few projects I've found that do some things similarly.

- [DataDash](https://github.com/keithknott26/datadash), a data visualization tool for the terminal.
- [Euoporie](https://github.com/joouha/euporie), a terminal interactive computing environment for
  Jupyter.
