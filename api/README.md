[![pulse logo](http://nano-assets.gopagoda.io/readme-headers/pulse.png)](http://nanobox.io/open-source#pulse)  
[![Build Status](https://travis-ci.org/nanopack/pulse.svg)](https://travis-ci.org/nanopack/pulse)

# Pulse

Pulse is a stat collecting and publishing service. It serves historical stats over an http api while live stats are sent to mist for live updates.

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

`*`: reserved query parameters are 'limit' and '[verb](https://docs.influxdata.com/influxdb/v0.13/query_language/functions)', all others act as filters  
`**`: reserved query parameters are '[verb](https://docs.influxdata.com/influxdb/v0.13/query_language/functions)', 'start', and 'stop', all others act as filters  

**note:** The API requires a token to be passed for authentication by default and is configurable at server start (`--token`). The token is passed in as a custom header: `X-AUTH-TOKEN`.  


## Usage Example:

#### get latest 'cpu_used' for 'web1' service
```sh
$ curl http://localhost:8080/latest/cpu_used?service=web1
# {"time":0,"value":0.2207}
```

#### get latest max 'cpu_used' for 'web1', 'web2', and 'web3' services (note: limit must be set)
```sh
$ curl "http://localhost:8080/latest/cpu_used?service=web1&service=web2&service=web3&limit=service&verb=max"
# {"time":0,"value":0.262}
```

#### get hourly 'cpu_used' for 'web1' service
```sh
$ curl http://localhost:8080/hourly/cpu_used?service=web1
# [{"time":1465419600000,"value":0.22331906098017212}]
```

#### get hourly max 'cpu_used' for 'web1', 'web2', and 'web3' services
```sh
$ curl "http://localhost:8080/hourly/cpu_used?service=web1&service=web2&service=web3&verb=max"
# [{"time":1465419600000,"value":0.22331906098017212}]
```
<sup>**note**: Multiple filters on hourly needs tested for accuracy</sup>

#### get daily averages 'cpu_used' for 'web1' service
```sh
$ curl http://localhost:8080/daily/cpu_used?service=web1
# {"15:0":0.22501574074074077,"15:15":0.22325925160697888,"15:30":0.22123160173160175}
```

#### get daily max 'cpu_used' for 'web1', 'web2', and 'web3' services for the last 3 days
```sh
$ curl "http://localhost:8080/daily/cpu_used?service=web1&service=web2&service=web3&verb=max&start=3d"
# [{"time":1465419600000,"value":0.22331906098017212}]
```
<sup>**note**: Multiple filters on daily needs tested for accuracy</sup>

#### get tags to filter by
```sh
$ curl -k -H "X-AUTH-TOKEN: secret" https://localhost:8080/tags
# ["host","service"]
```

#### get list of stats stored
```sh
$ curl http://localhost:8080/keys
# ["cpu_used","ram_used"]
```

#### add alert for cpu_used to trigger critical alert to localhost/alert if cpu_used is > 80 for 30s
```sh
$ curl http://localhost:8080/alerts -d '{
  "tags": {"host":"abcd"},
  "metric": "cpu_used",
  "level": "crit",
  "threshold": 80,
  "duration": "30s",
  "post": "http://127.0.0.1/alert"
}'
# {"tags":{"host":"abcd"},"metric":"cpu_used","level":"crit","threshold":80,"duration":"30s","post":"http://127.0.0.1/alert"}
```

#### delete alert for cpu_used to trigger critical alert to localhost/alert if cpu_used is > 80 for 30s
```sh
$ curl http://localhost:8080/alerts/cpu_used -X DELETE
# {"msg":"Success"}
```
<sup>**note**: Known bug regarding lack of unique alert id</sup>


[![open source](http://nano-assets.gopagoda.io/open-src/nanobox-open-src.png)](http://nanobox.io/open-source)
