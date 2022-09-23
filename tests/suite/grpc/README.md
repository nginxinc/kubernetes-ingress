The files here are generated using `helloworld.proto` from the grpc [repo](https://github.com/grpc/grpc-go/tree/master/examples/helloworld/helloworld).

To update the files run the following command:

```bash
python3 -m grpc_tools.protoc --proto_path=. --python_out=. --grpc_python_out=. helloworld.proto
```
