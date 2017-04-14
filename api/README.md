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

`*`: reserved query parameters is '[verb](https://docs.influxdata.com/influxdb/v0.13/query_language/functions)', all others act as filters  
`**`: reserved query parameters are 'backfill', '[verb](https://docs.influxdata.com/influxdb/v0.13/query_language/functions)', 'start', and 'stop', all others act as filters  

**note:** The API requires a token to be passed for authentication by default and is configurable at server start (`--token`). The token is passed in as a custom header: `X-AUTH-TOKEN`.  


## Usage Example:

#### get latest 'cpu_used' for 'web1' service
```sh
$ curl http://localhost:8080/latest/cpu_used?service=web1
# {"time":0,"value":0.2207}
```

#### get latest max 'cpu_used' for 'web1', 'web2', and 'web3' services
```sh
$ curl "http://localhost:8080/latest/cpu_used?service=web1&service=web2&service=web3&verb=max"
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

#### add alert for cpu_used to trigger critical alert to localhost/alert if cpu_used is > 0.80 for 30s
```sh
$ curl http://localhost:8080/alerts -d '{
  "tags": {"host":"abcd"},
  "metric": "cpu_used",
  "level": "crit",
  "threshold": "0.80",
  "duration": "30s",
  "post": "http://127.0.0.1/alert"
}'
# {"id":"785ba395-f2a0-47b3-b1cf-ee788dd0d5c8","tags":{"host":"abcd"},"metric":"cpu_percent","level":"crit","threshold":"0.80","duration":"30s","post":"http://127.0.0.1/alert"}
```

>Alert post body example:
>```
>{
>  "id": "[do.1] cpu_percent",
>  "message": "[do.1] cpu_percent is CRITICAL value:0.11818333333333335",
>  "details": "{&#34;Name&#34;:&#34;cpu_percent&#34;,&#34;TaskName&#34;:&#34;fe64d9d7-5b35-43f0-a57c-52a6ead3a361&#34;,&#34;Group&#34;:&#34;nil&#34;,&#34;Tags&#34;:null,&#34;ID&#34;:&#34;[do.1] cpu_percent&#34;,&#34;Fields&#34;:{&#34;mean_cpu_percent&#34;:0.11818333333333335},&#34;Level&#34;:&#34;CRITICAL&#34;,&#34;Time&#34;:&#34;2017-04-14T20:12:03.972677173Z&#34;,&#34;Message&#34;:&#34;[do.1] cpu_percent is CRITICAL value:0.11818333333333335&#34;}\n",
>  "time": "2017-04-14T20:12:03.972677173Z",
>  "duration": 0,
>  "level": "CRITICAL",
>  "data": {
>    "series": [
>      {
>        "name": "cpu_percent",
>        "columns": [
>          "time",
>          "mean_cpu_percent"
>        ],
>        "values": [
>          [
>            "2017-04-14T20:12:03.972677173Z",
>            0.11818333333333335
>          ]
>        ]
>      }
>    ]
>  }
>}
>
>```

#### delete alert for cpu_used to trigger critical alert to localhost/alert if cpu_used is > 0.80 for 30s
```sh
$ curl http://localhost:8080/alerts/785ba395-f2a0-47b3-b1cf-ee788dd0d5c8 -X DELETE
# {"msg":"Success"}
```

#### get alerts
```sh
$ curl http://localhost:8080/alerts
# [{"link":{"rel":"self","href":"/kapacitor/v1/tasks/36f9aaa7-689d-47b0-a08f-f9f6575548ee"},"id":"36f9aaa7-689d-47b0-a08f-f9f6575548ee","template-id":"","type":"batch","dbrps":[{"db":"statistics","rp":"one_day"}],"script":"batch\n    |query('''\n\t\tSELECT mean(ram_percent) AS mean_ram_percent\n\t\tFROM \"statistics\".\"one_day\".\"ram_percent\"\n\t\tWHERE \"host\" = 'do.1'\n\t''')\n        .period(1s)\n        .every(30s)\n    |alert()\n        .id('[do.1] ram_percent')\n        .message('{{ .ID }} is {{ .Level }} value:{{ index .Fields \"mean_ram_percent\" }}')\n        .crit(lambda: \"mean_ram_percent\" \u003e 2)\n        .post('http://127.0.0.1/alert')\n        .stateChangesOnly()\n        .log('/tmp/alerts.log')\n","vars":null,"dot":"digraph 36f9aaa7-689d-47b0-a08f-f9f6575548ee {\ngraph [throughput=\"0.00 batches/s\"];\n\nquery1 [avg_exec_time_ns=\"6.243805ms\" batches_queried=\"0\" points_queried=\"0\" query_errors=\"0\" ];\nquery1 -\u003e alert2 [processed=\"0\"];\n\nalert2 [alerts_triggered=\"0\" avg_exec_time_ns=\"0s\" crits_triggered=\"0\" infos_triggered=\"0\" oks_triggered=\"0\" warns_triggered=\"0\" ];\n}","status":"enabled","executing":true,"error":"","stats":{"task-stats":{"throughput":0},"node-stats":{"alert2":{"alerts_triggered":0,"avg_exec_time_ns":0,"collected":0,"crits_triggered":0,"emitted":0,"infos_triggered":0,"oks_triggered":0,"warns_triggered":0},"batch0":{"avg_exec_time_ns":0,"collected":0,"emitted":0},"query1":{"avg_exec_time_ns":6243805,"batches_queried":0,"collected":0,"emitted":0,"points_queried":0,"query_errors":0}}},"created":"2017-04-14T20:42:00.221022462Z","modified":"2017-04-14T20:42:00.221022462Z","last-enabled":"2017-04-14T20:42:00.221022462Z"},{"link":{"rel":"self","href":"/kapacitor/v1/tasks/c084588c-f2e9-447b-a67a-038a71b64639"},"id":"c084588c-f2e9-447b-a67a-038a71b64639","template-id":"","type":"batch","dbrps":[{"db":"statistics","rp":"one_day"}],"script":"batch\n    |query('''\n\t\tSELECT mean(cpu_percent) AS mean_cpu_percent\n\t\tFROM \"statistics\".\"one_day\".\"cpu_percent\"\n\t\tWHERE \"host\" = 'do.1'\n\t''')\n        .period(1s)\n        .every(30s)\n    |alert()\n        .id('[do.1] cpu_percent')\n        .message('{{ .ID }} is {{ .Level }} value:{{ index .Fields \"mean_cpu_percent\" }}')\n        .crit(lambda: \"mean_cpu_percent\" \u003e 1)\n        .post('http://api.nanobox.dev:8080/v1/triggers/1b2b22f3-62e1-4690-bc62-4cf3e156e20b/pull')\n        .stateChangesOnly()\n        .log('/tmp/alerts.log')\n","vars":null,"dot":"digraph c084588c-f2e9-447b-a67a-038a71b64639 {\ngraph [throughput=\"0.00 batches/s\"];\n\nquery1 [avg_exec_time_ns=\"3.637332ms\" batches_queried=\"0\" points_queried=\"0\" query_errors=\"0\" ];\nquery1 -\u003e alert2 [processed=\"0\"];\n\nalert2 [alerts_triggered=\"0\" avg_exec_time_ns=\"0s\" crits_triggered=\"0\" infos_triggered=\"0\" oks_triggered=\"0\" warns_triggered=\"0\" ];\n}","status":"enabled","executing":true,"error":"","stats":{"task-stats":{"throughput":0},"node-stats":{"alert2":{"alerts_triggered":0,"avg_exec_time_ns":0,"collected":0,"crits_triggered":0,"emitted":0,"infos_triggered":0,"oks_triggered":0,"warns_triggered":0},"batch0":{"avg_exec_time_ns":0,"collected":0,"emitted":0},"query1":{"avg_exec_time_ns":3637332,"batches_queried":0,"collected":0,"emitted":0,"points_queried":0,"query_errors":0}}},"created":"2017-04-14T19:58:36.752134021Z","modified":"2017-04-14T19:58:36.752134021Z","last-enabled":"2017-04-14T19:58:36.752134021Z"}]
```

#### get alert
```sh
$ curl http://localhost:8080/alerts/36f9aaa7-689d-47b0-a08f-f9f6575548ee
# {"link":{"rel":"self","href":"/kapacitor/v1/tasks/36f9aaa7-689d-47b0-a08f-f9f6575548ee"},"id":"36f9aaa7-689d-47b0-a08f-f9f6575548ee","template-id":"","type":"batch","dbrps":[{"db":"statistics","rp":"one_day"}],"script":"batch\n    |query('''\n\t\tSELECT mean(ram_percent) AS mean_ram_percent\n\t\tFROM \"statistics\".\"one_day\".\"ram_percent\"\n\t\tWHERE \"host\" = 'do.1'\n\t''')\n        .period(1s)\n        .every(30s)\n    |alert()\n        .id('[do.1] ram_percent')\n        .message('{{ .ID }} is {{ .Level }} value:{{ index .Fields \"mean_ram_percent\" }}')\n        .crit(lambda: \"mean_ram_percent\" \u003e 2)\n        .post('http://127.0.0.1/alert')\n        .stateChangesOnly()\n        .log('/tmp/alerts.log')\n","vars":null,"dot":"digraph 36f9aaa7-689d-47b0-a08f-f9f6575548ee {\ngraph [throughput=\"0.00 batches/s\"];\n\nquery1 [avg_exec_time_ns=\"6.243805ms\" batches_queried=\"0\" points_queried=\"0\" query_errors=\"0\" ];\nquery1 -\u003e alert2 [processed=\"0\"];\n\nalert2 [alerts_triggered=\"0\" avg_exec_time_ns=\"0s\" crits_triggered=\"0\" infos_triggered=\"0\" oks_triggered=\"0\" warns_triggered=\"0\" ];\n}","status":"enabled","executing":true,"error":"","stats":{"task-stats":{"throughput":0},"node-stats":{"alert2":{"alerts_triggered":0,"avg_exec_time_ns":0,"collected":0,"crits_triggered":0,"emitted":0,"infos_triggered":0,"oks_triggered":0,"warns_triggered":0},"batch0":{"avg_exec_time_ns":0,"collected":0,"emitted":0},"query1":{"avg_exec_time_ns":6243805,"batches_queried":0,"collected":0,"emitted":0,"points_queried":0,"query_errors":0}}},"created":"2017-04-14T20:42:00.221022462Z","modified":"2017-04-14T20:42:00.221022462Z","last-enabled":"2017-04-14T20:42:00.221022462Z"}
```


[![open source](http://nano-assets.gopagoda.io/open-src/nanobox-open-src.png)](http://nanobox.io/open-source)
