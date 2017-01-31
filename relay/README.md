[![pulse logo](http://nano-assets.gopagoda.io/readme-headers/pulse.png)](http://nanobox.io/open-source#pulse)   
[![Build Status](https://travis-ci.org/nanopack/pulse.svg)](https://travis-ci.org/nanopack/pulse)

# Pulse

Pulse is a stat collecting and publishing service. It serves historical stats over an http api while live stats are sent to mist for live updates.

## Client Example:

```go
package main

import (
  "fmt"
  "math/rand"
  "os"
  "os/exec"
  "strconv"
  "time"

  // if we want to see pulse client logs
  "github.com/jcelliott/lumber"

  pulse "github.com/nanopack/pulse/relay"
)

// address of pulse server
var address = "127.0.0.1:3000"

func main() {
  // because we want to see all pulse client logs
  lumber.Level(lumber.LvlInt("TRACE"))

  // returns system command getter for cpu-used
  var cpuGetter = func() float64 {
    raw, err := exec.Command("bash", "-c", "cat /proc/stat | awk '/cpu / {usage=($2+$4)/($2+$4+$5)} END {print usage}' | tr -d '\n'").Output()
    if err != nil {
      return -1
    }
    floatData, _ := strconv.ParseFloat(string(raw), 64)
    return floatData
  }

  // returns golang func for mock ram-used
  var ramGetter = rand.Float64

  // register a new client
  relay, err := pulse.NewRelay(address, "lester.tester")
  if err != nil {
    fmt.Printf("Unable to connect to pulse server %s\n", err)
    return
  }
  defer relay.Close()

  // add new cpu collector for a container
  cpuCollector := pulse.NewPointCollector(cpuGetter)
  if err := relay.AddCollector("cpu_used", []string{"","service:web1"}, cpuCollector); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }

  // a ram collector for the host
  ramCollector := pulse.NewPointCollector(ramGetter)
  if err := relay.AddCollector("ram_used", nil, ramCollector); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }

  // keep it running for a while
  time.Sleep(time.Minute * 30)
}
```

[![open source](http://nano-assets.gopagoda.io/open-src/nanobox-open-src.png)](http://nanobox.io/open-source)
