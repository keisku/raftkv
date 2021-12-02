# raftkv

This repository holds a simple distributed key-value store by using [hashicorp/raft](https://github.com/hashicorp/raft).
`raftkv` provides gRPC and HTTP APIs. Have a look [API Reference](./proto/v1/README.md).

## Usage

### Run containers

```shell
docker compose up
```

### Validation

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
