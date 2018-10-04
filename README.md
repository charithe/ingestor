Ingest Service
==============

This repository contains a minimal implementation of an HTTP server that parses a CSV file and updates a backend gRPC 
service with the records read from the file.

Please note that this implementation is not production ready. It uses plaintext connections instead of
TLS secured ones, does not have any metrics instrumentation, haven't been profiled or load tested, and the tests are 
quite basic. Given enough time, it can be further improved and made more robust.


How to build the project
-------------------------

If Docker Compose is installed on your machine, run the following command to start the services:

```shell
make launch
```

The above command starts the services and binds them to port 8080 and 8090. To test the ingestion, run the following
command:

```shell
curl -XPOST --data-binary @path/to/data.csv 'localhost:8080'
```

Alternatively, you can build the Docker images yourself by running `make docker`. This will create a Docker image
named `charithe/ingestor` tagged with the current Git hash. 

### Regenerating Protobuf Generated Code

In order to regenerate the protobuf code stubs, you'll need [Prototool](https://github.com/uber/prototool) installed
on your machine. After the tool is installed, run `make gen-proto` to regenerate the protobuf and gRPC code.




