# raftkv

:construction: still developing.... :construction:

This repository holds a simple distributed key-value store by using [hashicorp/raft](https://github.com/hashicorp/raft).

## Usage

Build a raftkv.

```shell
go build -o raftkv main.go
```

Start a leader server and create a cluster.

```shell
./raftkv --server-id=server0 \
--raft-addr=localhost:50000 \
--grpc-addr=localhost:50001 \
--grpcgw-addr=localhost:50002
```

Register a follower to a cluster.

```shell
./raftkv --server-id=server1 \
--join-addr=localhost:50000 \
--raft-addr=localhost:50100 \
--grpc-addr=localhost:50101 \
--grpcgw-addr=localhost:50102
```

```shell
./raftkv --server-id=server2 \
--join-addr=localhost:50000 \
--raft-addr=localhost:50200 \
--grpc-addr=localhost:50201 \
--grpcgw-addr=localhost:50202
```
