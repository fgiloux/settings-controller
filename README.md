# Pipeline-Service Settings Operator

WIP: currently transitioning from the controller-runtime example to the desired resources.

This operator is in charge of managing the settings of [kcp](https://github.com/kcp-dev/kcp) workspaces used for [Pipeline-Service](https://github.com/openshift-pipelines/pipeline-service).

## Description

Pipeline-Service offers an infrastructure for easily run Tekton Pipelines in a secured and isolated way. Therefore some restrictions need to be set on the workspaces than can consume the Pipeline-Service infrastructure.

- Quotas limit the amount of compute resources that can be consumed.
- NetworkPolicies restrict the access granted to the pods running the pipeline tasks.

## Getting Started

### Building the image

Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/settings-operator:tag
```

### Deploying to kcp

Deploy the operator to kcp with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/settings-operator:tag
```

### Uninstalling resources

To delete the resources from kcp:

```sh
make uninstall
```

### Undeploying the operator

Undeploy the operator from kcp:

```sh
make undeploy
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources until the desired state is reached. 

### Test It Out

1. Install the required resources into kcp:

```sh
make install
```

2. Run the operator (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions

If you are editing the API definitions, regenerate the manifests using:

```sh
make manifests apiresourceschemas
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
