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
You will get "golang" from followers. It means a leader propagates the value to followers.

```shell
curl -XGET 'localhost:50102/v1/get/programming_language'
curl -XGET 'localhost:50202/v1/get/programming_language'
```

Delete a key-value.
You will get "not found" errors from followers.

```shell
curl -XDELETE 'localhost:50002/v1/delete/programming_language'
```

```shell
curl -XGET 'localhost:50102/v1/get/programming_language'
curl -XGET 'localhost:50202/v1/get/programming_language'
```

```json
{"code":5, "message":"not found", "details":[]}
{"code":5, "message":"not found", "details":[]}
```

If you try to set or delete a key-value to a follower, you will get an error.
`localhost:50102` is an address for a follower at this time and you can replace it with `localhost:50201` too.

See the [docker-compose.yaml](./docker-compose.yaml) in detail.

```shell
curl -XPOST 'localhost:50102/v1/set' \
--data-raw '{
    "key": "programming_language",
    "value": "golang"
}'
curl -XDELETE 'localhost:50102/v1/delete/programming_language'
```

```json
{"code":13, "message":"non-leader can't set key=programming_language, value=golang", "details":[]}
{"code":13, "message":"non-leader can't delete a value associated to programming_language", "details":[]}
```
