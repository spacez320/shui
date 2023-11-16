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
  -q string
    	Query to execute. (default "whoami")
  -s	Don't output anything to a console.
  -t int
    	Number of query executions. -1 for continuous. (default 1)
```

### Examples

```sh
# Execute `whoami` once, printing results to the console.
cryptarch -q 'whoami'

# Execute `uptime` continuously, printing results to the console.
cryptarch -q 'uptime' -t -1

# Do the same thing, but silently in the background.
cryptarch -q 'uptime' -s -t -1 &
```
