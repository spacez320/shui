cryptarch
=========

Cryptarch is a tool that can be used to run commands and parse their responses
over a time range.

This project is in the toy phase and might not do much or work very well.

Usage
-----

```
Usage of ./cryptarch:
  -d int
        Delay between queries (seconds). (default 3)
  -l string
        Log level. (default "error")
  -m int
        Mode to execute in.
  -p string
        Port for RPC. (default "12345")
  -q value
        Query to execute. (default 0xc0000145f0)
  -r int
        Result mode to display. (default 2)
  -s    Don't output anything to a console.
  -t int
        Number of query executions. -1 for continuous. (default 1)
```

Cryptarch executes in one of two modes.

- **Query** mode is the default and is for querying.
- **Read** mode is for interacting with an already running Cryptarch instance.

### Examples

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
