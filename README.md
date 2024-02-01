cryptarch
=========

Cryptarch is a tool that can be used to run shell commands and observe their responses over time. It
is meant to aid system administrators, platform engineers, or anyone who spends time observing
things in a CLI.

- It's like `watch` except it can draw graphs, tables, etc.
- It's like a simplified `Prometheus` except you can run it directly in your console.

This project is in an active, "alpha" development phase, but should still be useable.

Setup
-----

Binaries are available from [the releases page](https://github.com/spacez320/cryptarch/releases).

Usage
-----

Cryptarch expects a **Query** to gather data with (e.g. a CLI command like `whoami` or `ps aux | wc
-l`). Queries produce **Results** that are stored as a time series.

### Modes

Cryptarch has **"modes"**.

**Query mode** is the default and is for running shell commands.

![Demo of query mode](https://raw.githubusercontent.com/spacez320/cryptarch/53cf72e8a181a911988d2ec45bd0cab6ca653cc6/media/query-mode.gif)

**Profile mode** is like Query mode except specialized for inspecting systems or processes.

![Demo of profile mode](https://raw.githubusercontent.com/spacez320/cryptarch/53cf72e8a181a911988d2ec45bd0cab6ca653cc6/media/process-mode.gif)

### Displays

Cryptarch also has **"displays"** that determine how data is presented.

**Raw display** and **Stream display** just presents incoming data, the latter being within
Cryptarch's interactive window. The examples above use stream displays.

**Table display** will parse results into a table.

![Demo of table display](https://raw.githubusercontent.com/spacez320/cryptarch/53cf72e8a181a911988d2ec45bd0cab6ca653cc6/media/table-display.gif)

**Graph display** will target a specific field in a result and graph it.

![Demo of graph display](https://raw.githubusercontent.com/spacez320/cryptarch/53cf72e8a181a911988d2ec45bd0cab6ca653cc6/media/graph-display.gif)

### More Examples

> The examples below have been tested on `GNU bash, version 5.2.15(1)-release`.

```sh
# See help.
cryptarch -h

# Execute `whoami` once, printing results to the console.
cryptarch -q 'whoami'

# Execute `uptime` continuously, printing results to the console.
cryptarch -q 'uptime' -t -1

# Get the size of an NVME used space and output it to a table.
```

Future
------

I've been adding planned work into [project issues](https://github.com/spacez320/cryptarch/issues)
and [project milestones](https://github.com/spacez320/cryptarch/milestone/1)--take a look there to
see what's coming.

Planned improvements include things like:

- Background execution and persistent results.
- Ability to perform calculations on streams of data, such as aggregates, rates, or quantile math.
- Better text result management, such as diff'ing.
- Export data to external systems, such as Prometheus.
- More detailed graph display modes.
