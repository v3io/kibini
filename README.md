Kibini is a logging parser to help us work with platform JSON until we have a UI. I know there are a few tools out there but I wanted one with the following abilities:
* Easy to install from anywhere - kibini is written in Go and its self contained, install-less binary is deployed on a public S3 bucket
* Parse multiple log files in one command - kibini parses a full CI run log (> 150K log records across all service logs) in under a second
* Tailing - kibini can tail files
* Combine multiple logs to a single log sorted by time - kibini supports this using a heuristic so that combining tailed logs is also supported
* Familiar output - kibini outputs an stdout-like log and pretty prints arguments. It also supports colors if you output to stdout

Kibini's defaults all assume the logs are in `cwd`, so it's best to run it from the log directory. If you want to run it on some transient machine, just download kibini straight the log dir. On long living machines, put it (or symlink it) in /usr/bin/kibini (/usr/local/bin/kibini on OSX) and run it in the log dir.

## Install
TBD

## Running
All these examples assume you're running it in the log directory. You can override paths using `--input-path` and `--output-path`.

#### Parse all log files in current directory
Will generate a *.log.fmt file for each *.log file.

`kibini`

#### Parse only container provisioning and shutdown logs (including their adapters)
--services and --no-services accept regular expressions.

`kibini --services container_provisioning|shutdown`

#### Parse all logs except for adapter logs and tail
`kibini -f --no-services adapter`

#### Parse all log files, merge them sorted by time and output only to stdout, tailing all (stdout forces --output-mode single)
`kibini -f --stdout`

#### Parse all logs, merge them sorted by time and output to to cwd/merged.log.fmt (you can change the output name by passing --output-path <file name>
`kibini --output-mode single`
