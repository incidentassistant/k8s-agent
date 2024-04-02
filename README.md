# Incident Assistant Kubernetes Agent

Incident Assistant is a Kubernetes agent designed to assist in incident management by watching for events within a Kubernetes cluster and handling them accordingly.

## Features

- **Dynamic Resource Watching**: Watches for events on dynamically discovered Kubernetes resources.
- **Event Handling**: Processes events for added, modified, and deleted resources.
- **Change Detection**: Computes and logs the differences between the old and new states of modified objects.
- **Protobuf Service**: Includes a protobuf service definition for emitting image change events.
- **Deployment Resources**: Provides Kubernetes deployment manifests for easy setup.

## Prerequisites

- A running Kubernetes cluster or Minikube for local development.
- Docker for building the container image.

## Installation

To deploy the Incident Assistant controller to your Kubernetes cluster, follow these steps:

1. Build the Docker image:
   ```sh
   docker build -t incidentassistant-controller .
   ```

2. Load the image into Minikube (if using Minikube):
   ```sh
   minikube image load incidentassistant-controller
   ```

3. Apply the Kubernetes manifests:
   ```sh
   kubectl apply -f install.yaml
   ```

## Development

### Building the Binary

To build the binary for the Incident Assistant AI controller, run:

```sh
go build -o controller ./cmd/controller
```

### Generating Protobuf Files

To generate Go files from the protobuf definition, run:

```sh
protoc --go_out=. --go-grpc_out=. ./proto/imagechange/imagechange.proto
```

## Contributing

Contributions to the Incident Assistant AI project are welcome! Please follow the standard GitHub pull request workflow to submit your changes.

## License

This project is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgments

This project uses the following open-source packages:

- [client-go](https://github.com/kubernetes/client-go)
- [gjson](https://github.com/tidwall/gjson)
- [jsondiff](https://github.com/wI2L/jsondiff)
