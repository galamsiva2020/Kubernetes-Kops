
<!--- This file is automatically generated by make gen-cli-docs; changes should be made in the go CLI command code (under cmd/kops) -->

## kops toolbox dump

Dump cluster information

### Synopsis

Displays cluster information.  Includes information about cloud and Kubernetes resources.

```
kops toolbox dump [CLUSTER] [flags]
```

### Examples

```
  # Dump cluster information
  kops toolbox dump --name k8s-cluster.example.com
```

### Options

```
      --dir string           Target directory; if specified will collect logs and other information.
  -h, --help                 help for dump
  -o, --output string        Output format.  One of json or yaml (default "yaml")
      --private-key string   File containing private key to use for SSH access to instances (default "~/.ssh/id_rsa")
      --ssh-user string      The remote user for SSH access to instances (default "ubuntu")
```

### Options inherited from parent commands

```
      --add_dir_header                   If true, adds the file directory to the header of the log messages
      --alsologtostderr                  log to standard error as well as files
      --config string                    yaml config file (default is $HOME/.kops.yaml)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --log_file string                  If non-empty, use this log file
      --log_file_max_size uint           Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
      --name string                      Name of cluster. Overrides KOPS_CLUSTER_NAME environment variable
      --one_output                       If true, only write logs to their native severity level (vs also writing to each lower severity level)
      --skip_headers                     If true, avoid header prefixes in the log messages
      --skip_log_headers                 If true, avoid headers when opening log files
      --state string                     Location of state storage (kops 'config' file). Overrides KOPS_STATE_STORE environment variable
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [kops toolbox](kops_toolbox.md)	 - Miscellaneous, infrequently used commands.
