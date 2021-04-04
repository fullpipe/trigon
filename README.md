# trigon

Trigon connects to MySQL bin-log stream and processes it with different pipes

## Install

```bash
go get -u github.com/fullpipe/trigon
```

## Usage

```bash
trigon -config=trigon.yaml
```

## Configuration with examples

```yaml
# trigon.yaml
input:
  host: 127.0.0.1:3320 # mysql host
  user: root
  password: root
triggers:
  - name: "Post webhook after 1000 user creations"
    pipes:
      - ["table_filter", ["user"]]
      - ["action_filter", ["insert"]]
      - ["every", 1000]
      - ["debounce", 5000]
      - ["webhook", { url: https://hookb.in/mZaZlXYLpjueqq710Ypk, method: POST }]

  - name: "Start data aggregation on leads updates"
    pipes:
      - ["table_filter", ["lead"]]
      - ["action_filter", ["insert", "update"]]
      - ["every", 10000]
      - ["debounce", 3600000] # debounce for one hour
      - ["log", { prefix: "Calculate leads aggregation" }]
      - ["amqp", { dsn: "amqp://guest:guest@localhost:5672/", exchange: "lead", routing_key: "lead.aggregate" }]

  - name: "Try every pipe on transactions"
    pipes:
      - ["log", { prefix: "All in one start: ", output: "file", "path": "./log.log" }]
      - ["table_filter", ["transaction"]]
      - ["action_filter", ["insert", "update", "delete"]]
      - ["every", 2]
      - ["debounce", 5000]
      - ["webhook", { url: https://hookb.in/mZaZlXYLpjueqq710Ypk, method: POST }]
      - ["amqp", { dsn: "amqp://guest:guest@localhost:5672/", exchange: "test1", routing_key: "order" }]
      - ["log", { prefix: "All in one end: " }]
```



## TODO
- [x] yaml config
    - [x] input
    - [x] triggers
        - [x] pipes
        - [x] output
- [ ] tests
- [ ] pipes
    - [x] log
      - [ ] formats
    - [x] action filter
    - [x] table filter
    - [x] every nth
    - [x] debounce
    - [x] amqp
    - [x] webhook
      - [ ] auth
    - [ ] expr filter, use https://github.com/antonmedv/expr for filtering
