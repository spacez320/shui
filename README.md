cryptarch
=========

Cryptarch is a tool that can be used to run shell commands and observe their responses over time. It
is meant to aid system administrators, platform engineers, or anyone who spends time observing
things in a CLI.

- It's like `watch` except it can draw graphs, tables, etc.
- It's like a simplified `Prometheus` except you can run it directly in your console.

This project is in an active, "alpha" development phase, but should still be useable.

Usage
-----

Cryptarch expects a **Query** to gather data with (e.g. a CLI command like `whoami` or `ps aux | wc
-l`). Queries produce **Results** that are stored as a time series.

### Modes

Cryptarch has **Modes"**.

**Query mode** is the default and is for running shell commands.

<<example>>

**Profile mode** is like Query mode except specialized for inspecting systems or processes.

<<example>>

### Displays

Cryptarch also has **"Displays"** that determine how data is presented.

**Stream display** just presents incoming data.

<<example>>

**Table display** will parse results into a table.

<<example>>

**Graph display** will target a specific field in a result and graph it.

<<example>>

### More Examples

> The examples below have been tested on `GNU bash, version 5.2.15(1)-release`.

```sh
# Execute `whoami` once, printing results to the console.
cryptarch -q 'whoami'

# Execute `uptime` continuously, printing results to the console.
cryptarch -q 'uptime' -t -1

# Get the size of an NVME used space.
cryptarch -q 'df -h | grep nvme0n1p2 | awk '\''{print $3}'\'''

# Do the same thing, but silently in the background. Then retrieve results.
cryptarch -q 'uptime' -s -t -1 &
cryptarch -m 1
```

Future
------

I've been adding planned work into [project issues](https://github.com/spacez320/cryptarch/issues)
and [project milestones](https://github.com/spacez320/cryptarch/milestone/1)--take a look there to
see what's coming.

Big planned improvements include things like:

- Ability to perform calculations on streams of data, such as aggregates, rates, or quantile math.
- Better text result management, such as diff'ing.
- Export data to external systems, such as Prometheus.
- More detailed graph display modes.
