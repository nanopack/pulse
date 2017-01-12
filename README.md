[![pulse logo](http://nano-assets.gopagoda.io/readme-headers/pulse.png)](http://nanobox.io/open-source#pulse)  
[![Build Status](https://travis-ci.org/nanopack/pulse.svg)](https://travis-ci.org/nanopack/pulse)

# Pulse

Pulse is a stat collecting and publishing service. It serves historical stats over an http api while live stats are sent to mist for live updates.


## Usage

Simply running `pulse -s` will start pulse with the default config options.  
`pulse -h` or `pulse --help` will show more detailed usage and config options:

```
Usage:
  pulse [flags]

Flags:
  -a, --aggregate-interval int         Interval at which stats are aggregated (default 15)
  -b, --beat-interval int              Heartbeat frequency (seconds) (default 30)
  -c, --config-file string             Config file location for server
  -C, --cors-allow string              Sets the 'Access-Control-Allow-Origin' header (default "*")
  -H, --http-listen-address string     Http listen address (default "127.0.0.1:8080")
  -i, --influx-address string          InfluxDB server address (default "http://127.0.0.1:8086")
  -I, --insecure                       Run insecure (default true)
  -k, --kapacitor-address string       Kapacitor server address (http://127.0.0.1:9092)
  -l, --log-level string               Level at which to log (default "INFO")
  -m, --mist-address string            Mist server address
  -M, --mist-token string              Mist server token
  -p, --poll-interval int              Interval to request stats from clients (default 60)
  -r, --retention int                  Number of weeks to store aggregated stats (default 1)
  -s, --server                         Run as server
  -S, --server-listen-address string   Server listen address (default "127.0.0.1:3000")
  -t, --token string                   Security token (recommend placing in config file) (default "secret")
  -v, --version                        Print version info and exit
```

### Config File Options:
```json
{
  "server": true,
  "server-listen-address": "127.0.0.1:3000",
  "http-listen-address": "127.0.0.1:8080",
  "influx-address": "http://127.0.0.1:8086",
  "kapacitor-address": "http://127.0.0.1:9092",
  "insecure": true,
  "mist-address": "",
  "mist-token": "",
  "log-level": "info",
  "cors-allow": "*",
  "token": "secret",
  "poll-interval": 60,
  "aggregate-interval": 15,
  "beat-interval": 30,
  "retention": 12
}
```


## API

| Route | Description | Output |
| --- | --- | --- |
| **GET** /keys | Returns list of stats being recorded | string array |
| **GET** /tags | Returns list of filterable tags | string array |
| **GET** /latest/{stat}* | Returns latest stat (averages if multiple filters applied) | json stat object |
| **GET** /hourly/{stat}** | Returns hourly averages for stat | json array of stat objects |
| **GET** /daily/{stat}** | Returns average for stat at the same daily time | string map |

**ALERTS** (requires "kapacitor-address" to be configured)  

| Route | Description | Payload | Output |
| --- | --- | --- | --- |
| **POST** /alerts | Add a kapacitor alert | json alert object | json alert object |
| **PUT** /alerts | Update a kapacitor alert | json alert object | json alert object |
| **DELETE** /alerts/{alert} | Delete a kapacitor alert | nil | success message |

`*`: reserved query parameters is 'verb', all others act as filters  
`**`: reserved query parameters are 'backfill', 'verb', 'start', and 'stop', all others act as filters  

**note:** The API requires a token to be passed for authentication by default and is configurable at server start (`--token`). The token is passed in as a custom header: `X-AUTH-TOKEN`.  

For examples, see [the api's readme](api/README.md).


## Data Types
### Stat Object
json:
```json
{
  "time": 1465419600,
  "value": 0.75
}
```

Fields:
- **time**: Unix epoch timestamp of stat
- **value**: Numeric value of stat

### Alert Object
json:
```json
{
  "tags": {"host":"abc"},
  "metric": "cpu_used",
  "level": "crit",
  "threshold": 80,
  "duration": "30s",
  "post": "http://127.0.0.1/alert"
}
```

Fields:
- **tags**: Populates the WHERE
- **metric**: Stat to track
- **level**: Alert level (info, warn, crit)
- **threshold**: Limit that alert is triggered at
- **duration**: How far back to average (5m)
- **post**: Api to hit when alert is triggered


## Relay

A pulse relay is a service that connects to pulse and advertises stats that are available for collection. A relay implementation is available in the pulse project and can be embedded in other projects.  

For an [**example**](relay/README.md), look in the README for relay.

### TCP pulse api
The TCP api used to communicate between the pulse server and a relay is simple and is designed to be human readable and debuggable. It is newline delimited.

| Command | Description | Response |
| --- | --- | --- |
| `id {id}` | **Must** be the first command to be run, identifies the client to the server | `ok` |
| `add {name}` | Exposes a stat that can be collected by the server | `ok` |
| `remove {name}` | Removes a stat previously exposed to the server | `ok` |


### TCP relay api
| Command | Description | Response |
| --- | --- | --- |
| `get {tag,tag2}` | Request a list of stats corrosponding to the list of tags passed in | `got {tag:value}` |
| `flush` | Clear all current values from the stat collectors | `ok` |
| `override {duration} {tag:interval}` | for `duration` seconds, bump the collection interval from the default to `interval` for each `tag:interval` | `ok` |

#### Notes
- If an override is specified for a stat, and a new machine comes online and connects, that override is **NOT** honored.
- Pulse server does not actively connect to servers to have stats pushed to it, rather, it waits for stat collecting machines to connect and then requests certain stats on specific intervals.


## Contributing

Contributions to the pulse project are welcome and encouraged. Pulse is a [Nanobox](https://nanobox.io) project and contributions should follow the [Nanobox Contribution Process & Guidelines](https://docs.nanobox.io/contributing/).

#### TODO

## Licence

Mozilla Public License Version 2.0

[![open source](http://nano-assets.gopagoda.io/open-src/nanobox-open-src.png)](http://nanobox.io/open-source)
