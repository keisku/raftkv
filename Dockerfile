FROM golang:1.17 as builder
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o app main.go

FROM alpine:3.13
WORKDIR /
COPY --from=builder /workspace/app .

# Create a raftkv user and group first so the IDs get set the same way, even as
# the rest of this may change over time.
RUN addgroup raftkv && \
	adduser -S -G raftkv raftkv

# The /raftkv/data dir is used by Consul to store state.
RUN mkdir -p /raftkv.d/data && \
	chown -R raftkv:raftkv /raftkv.d

# Expose the raftkv data directory as a volume since there's mutable state in there.
VOLUME /raftkv.d/data

CMD ["/app"]
