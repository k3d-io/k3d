# Go package for working with a system's hostsfile
[![codecov](https://codecov.io/gh/goodhosts/hostsfile/branch/main/graph/badge.svg?token=BJQH16QQEH)](https://codecov.io/gh/goodhosts/hostsfile)
[![Go Reference](https://pkg.go.dev/badge/github.com/goodhosts/hostsfile.svg)](https://pkg.go.dev/github.com/goodhosts/hostsfile)

Reads the content of a file in the [hosts format](https://en.wikipedia.org/wiki/Hosts_(file)) into go structs for easy manipulation in go programs. When all changes are complete you can `Flush` the hosts file back to disk to save your changes. Supports an indexing system on both ips and hosts for quick management of large hosts files.    

## Simple Usage
Simple usage reading in your system's hosts file and adding an entry for the ip `192.168.1.1` and the host `my-hostname`

```go
package main

import (
	"log"
	
	"github.com/goodhosts/hostsfile"
)

func main() {
    hosts, err := hostsfile.NewHosts()
    if err != nil {
        log.Fatal(err.Error())
    }
    if err := hosts.Add("192.168.1.1", "my-hostname"); err != nil {
        log.Fatal(err.Error())
    }
    if err := hosts.Flush(); err != nil {
        log.Fatal(err.Error())
    }
}
```

### Other Usage
Read in a hosts file from a custom location which is not the system default, this is useful for tests or systems with non-standard hosts file locations.
```
hosts, err := hostsfile.NewCustomHosts("./my-custom-hostsfile")
```

Use `Add` to put an ip and host combination in the hosts file
```
err := hosts.Add("192.168.1.1", "my-hostname")
```

`Add` is variadic and can take multiple hosts to add for the same ip
```
err := hosts.Add("192.168.1.1", "my-hostname", "another-hostname")
```

Use `Remove` to drop an ip and host combination from the hosts file
```
err := hosts.Remove("192.168.1.1", "my-hostname")
```

`Remove` is variadic and can take multiple hosts to remove from the same ip
```
err := hosts.Remove("192.168.1.1", "my-hostname", "another-hostname")
```

Flush the hosts file changes back to disk
```
err := hosts.Flush()
```
