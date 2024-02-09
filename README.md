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

Binaries are available from [the releases page](https://github.com/spacez320/cryptarch/releases).

Usage
-----

Cryptarch expects a **query** to gather data with (e.g. a CLI command, a process ID, etc.). Queries
produce **results** that are parsed automatically and stored as a time series.

### Modes

Cryptarch has **"modes"** that determine what type of query should be provided.

**Query mode** is the default and is for running shell commands.

![Demo of query mode](https://raw.githubusercontent.com/spacez320/cryptarch/master/media/query-mode.gif)

**Profile mode** is like Query mode except specialized for inspecting systems or processes.

![Demo of profile mode](https://raw.githubusercontent.com/spacez320/cryptarch/master/media/profile-mode.gif)

### Displays

Cryptarch also has **"displays"** that determine how data is presented.

**Raw display** and **stream display** simply presents incoming data, the latter being within
Cryptarch's interactive window. The examples above use stream displays.

**Table display** will parse results into a table.

![Demo of table display](https://raw.githubusercontent.com/spacez320/cryptarch/master/media/table-display.gif)

**Graph display** will target a specific field in a result and graph it.

![Demo of graph display](https://raw.githubusercontent.com/spacez320/cryptarch/master/media/graph-display.gif)

### Persistence

Cryptarch, by default, will store results and load them when re-executing the same query. Storage is
located in the user's cache directory.

See: <https://pkg.go.dev/os#UserCacheDir>

### More Examples

> The examples below have been tested on `GNU bash, version 5.2.15(1)-release`.

```sh
# See help.
cryptarch -h

# Execute `whoami` once, printing results to the console and waiting for a user to `^C`.
cryptarch -q 'whoami'

# Execute `uptime` continuously, printing results to the console, without using persistence.
cryptarch -q 'uptime' -t -1 -e=false

# Get the size of an NVME used space and output it to a table.
cryptarch -q 'df -h | grep nvme0n1p2 | awk '\''{print $3}'\''' -r 3 -v "NVME Used Space" -t -1
```

Future
------

I've been adding planned work into [project issues](https://github.com/spacez320/cryptarch/issues)
and [project milestones](https://github.com/spacez320/cryptarch/milestone/1)--take a look there to
see what's coming or make suggestions.

Planned improvements include things like:

- Background execution and persistent results.
- Ability to perform calculations on streams of data, such as aggregates, rates, or quantile math.
- Better text result management, such as diff'ing.
- Export data to external systems, such as Prometheus.
- More detailed graph display modes.

Similar Projects
----------------

- [DataDash](https://github.com/keithknott26/datadash), a data visualization tool for the terminal.
