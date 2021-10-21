# gRPC client

From https://grpc.io/docs/quickstart/go.html. Modified to establish TLS connections.

## Compile for MacOS in Docker

```
$ make
```

## Run

```
$ ./grpc_client
```

By default, it connects to grpc.example.com:8443 over TLS. To override the destination address, use the `-address` flag:

```
$ ./grpc_client -address your.example.com:8443