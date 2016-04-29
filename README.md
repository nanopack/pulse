[![pulse logo](http://nano-assets.gopagoda.io/readme-headers/pulse.png)](http://nanobox.io/open-source#pulse)   
[![Build Status](https://travis-ci.org/nanopack/pulse.svg)](https://travis-ci.org/nanopack/pulse)

# Pulse

Pulse is a stat collecting and publishing service. It serves historical stats over an http api while live stats are sent to mist for live updates.

## Status

Complete/Experimental


## Server
### Usage:
- `pulse [flags]`

### Flags:
```
  -a, --aggregate_interval=15: Interval at which stats are aggregated
  -c, --config_file="": Config file location for server
  -H, --http_listen_address="127.0.0.1:8080": Http listen address
  -i, --influx_address="127.0.0.1:8086": InfluxDB server address
  -l, --log_level="INFO": Level at which to log
  -m, --mist_address="": Mist server address
  -p, --poll_interval=60: Interval to request stats from clients
  -s, --server[=false]: Run as server
  -S, --server_listen_address="127.0.0.1:3000": Server listen address
  -t, --token="secret": Security token (recommend placing in config file)
```

### Config File Options:
```json
{
  "server": false,
  "server_listen_address": "127.0.0.1:3000",
  "http_listen_address": "127.0.0.1:8080",
  "influx_address": "127.0.0.1:8086",
  "mist_address": "",
  "log_level": "INFO",
  "token": "secret",
  "poll_interval": 60,
  "aggregate_interval": 15
}
```

## Relay

A pulse relay is a service that connects to pulse and advertises stats that are available for collection. A relay implementation is available in the pulse project and can be embedded in other projects.  
For an [**example**](relay/README.md), look in the README for relay

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


## Routes

| Url | Description | Payload | Output |
| --- | --- | --- | --- |
| `/services/{service}/stats/{stat}/hourly` | Grab hourly averages for the last day | nil | `[{"time":14463123000,"value":0.124}]` |
| `/services/{service}/stats/{stat}/daily_peaks` | Grab combined 15 minute averages for the last week | nil | `{"16:15":0.1}`

#### Example:
Get 'ram_used' stat for 'web1' service
```
$ curl -k -H "X-NANOBOX-TOKEN: secret" https://127.0.0.1:8080/services/web1/stats/ram_used/hourly
[{"time":1455665400000,"value":0.43448749999999997},{"time":1455666300000,"value":0.43753846153846154},{"time":1455667200000,"value":0.4414133333333333},{"time":1455667200000,"value":0.7366},{"time":1455668100000,"value":0.45486999999999994}]
```
Get 15 min average of 'ram_used' stat for 'web1' service
```
$ curl -k -H "X-NANOBOX-TOKEN: secret" https://127.0.0.1:8080/services/web1/stats/ram_used/daily_peaks
{"16:30":0.43448749999999997,"16:45":0.43753846153846154,"17:0":1.1780133333333334,"17:15":0.45486999999999994}
```

## Notes
- If an override is specified for a stat, and a new machine comes online and connects, that override is **NOT** honored.
- Pulse server does not actively connect to servers to have stats pushed to it, rather, it waits for stat collecting machines to connect and then requests certain stats on specific intervals.

### Contributing

Contributions to the pulse project are welcome and encouraged. Pulse is a [Nanobox](https://nanobox.io) project and contributions should follow the [Nanobox Contribution Process & Guidelines](https://docs.nanobox.io/contributing/).

**todo**:  
- uniq the list of tags and fields in influx/influx.go:138 and possibly apply to api/request.go:24  
- there may be a bug with continuous queries aggregating by host and service rather than just service  
- there may also be a bug getting hourly stats that returns all aggregated stats  
- there may also be a bug with daily peaks that adds the stat from different hosts' (maybe it needs to divide by number of hosts/instances with that stat)  

### Licence

Mozilla Public License Version 2.0

[![open source](http://nano-assets.gopagoda.io/open-src/nanobox-open-src.png)](http://nanobox.io/open-source)
