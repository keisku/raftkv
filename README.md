# raftkv

This repository holds A simple distributed key-value store by using [hashicorp/raft](https://github.com/hashicorp/raft).

## Usage

Build a raftkv.

```shell
go build -o raftkv ./cmd/raftkv/main.go
```

Start a leader server and create a cluster.

```shell
export LOG_LEVEL=1 \
SERVER_ID=server0 \
RAFT_ADDR=localhost:50000 \
GRPC_ADDR=:50001 \
GRPC_GATEWAY_ADDR=:50002 && ./raftkv
```

Join a follower to a cluster.

```shell
export LOG_LEVEL=1 \
SERVER_ID=server1 \
JOIN_ADDR=localhost:50001 \
RAFT_ADDR=localhost:50100 \
GRPC_ADDR=:50101 \
GRPC_GATEWAY_ADDR=:50102 && ./raftkv
```

```shell
export LOG_LEVEL=1 \
SERVER_ID=server2 \
JOIN_ADDR=localhost:50001 \
RAFT_ADDR=localhost:50200 \
GRPC_ADDR=:50201 \
GRPC_GATEWAY_ADDR=:50202 && ./raftkv
```
