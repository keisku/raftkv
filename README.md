# raftkv

:construction: still developing.... :construction:

This repository holds a simple distributed key-value store by using [hashicorp/raft](https://github.com/hashicorp/raft).

## Usage

### Run binaries

Build a raftkv.

```shell
go build -o raftkv main.go
```

Start a leader server and create a cluster.

```shell
./raftkv --server-id=server0 \
--raft-addr=localhost:50000 \
--grpc-addr=localhost:50001 \
--grpcgw-addr=localhost:50002 \
--dir=raftkv.d
```

Register a follower to a cluster.

```shell
./raftkv --server-id=server1 \
--join-addr=localhost:50001 \
--raft-addr=localhost:50100 \
--grpc-addr=localhost:50101 \
--grpcgw-addr=localhost:50102 \
--dir=raftkv.d
```

```shell
./raftkv --server-id=server2 \
--join-addr=localhost:50001 \
--raft-addr=localhost:50200 \
--grpc-addr=localhost:50201 \
--grpcgw-addr=localhost:50202 \
--dir=raftkv.d
```

### Run containers

```shell
docker compose up
```

### Test

Set a key-value to a leader.

```shell
curl -XPOST 'localhost:50002/v1/set' \
--data-raw '{
    "key": "programming_language",
    "value": "golang"
}'
```

Get a value from a follower.

```shell
curl -XGET 'localhost:50102/v1/get/programming_language'
curl -XGET 'localhost:50202/v1/get/programming_language'
```

You can get "golang" from followers. It means a leader propagates the value to followers.
