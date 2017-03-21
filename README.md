# process
Managing processes with Go

[![GoDoc](https://godoc.org/github.com/nightlyone/process?status.svg)](https://godoc.org/github.com/nightlyone/process)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/nightlyone/process)
[![Build Status](https://secure.travis-ci.org/nightlyone/process.png)](https://travis-ci.org/nightlyone/process)

# example usage
```go
package main
import (
	"os/exec"
	"time"

	"github.com/nightlyone/process"
)

func main() {
	cmd := exec.Command("true")
	group, err := process.Background(cmd)
	if err != nil {
		panic(err)
	}
	group.Terminate(1 * time.Second)
}
```
# LICENSE
BSD

# build and install
Install [Go 1][3], either [from source][4] or [with a prepackaged binary][5].

[3]: http://golang.org
[4]: http://golang.org/doc/install/source
[5]: http://golang.org/doc/install

Then run

    go get github.com/nightlyone/process

# NOTE
 This package will only work on UNIX based systems
