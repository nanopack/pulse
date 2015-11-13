[![pulse logo](http://nano-assets.gopagoda.io/readme-headers/pulse.png)](http://nanobox.io/open-source#pulse)
 [![Build Status](https://travis-ci.org/nanopack/pulse.svg)](https://travis-ci.org/nanopack/pulse)

# Pulse

Pulse is a stat collecting and publishing service. It serves historical stats over an http api, and live stats are sent to mist for live updates.

## Status

Complete/Experimental

## Relay

A pulse relay is a service that connects to pulse and advertises stats that are available for collection. A relay implementation is available in the pulse project and can be embedded in other projects.


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

## Notes
- If an override is specified for a stat, and a new machine comes online and connects, that override is **NOT** honored.
- Pulse server does not actively connect to servers to have stats pushed to it, rather, it waits for stat collecting machines to connect and then requests certain stats on specific intervals.

### Contributing

Contributions to the pulse project are welcome and encouraged. Pulse is a [Nanobox](https://nanobox.io) project and contributions should follow the [Nanobox Contribution Process & Guidelines](https://docs.nanobox.io/contributing/).

### Licence

Mozilla Public License Version 2.0

[![open source](http://nano-assets.gopagoda.io/open-src/nanobox-open-src.png)](http://nanobox.io/open-source)
