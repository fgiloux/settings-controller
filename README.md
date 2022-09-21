# Pipeline Service Settings Operator

This operator is in charge of managing the settings of [kcp](https://github.com/kcp-dev/kcp) workspaces used for [Pipeline Service](https://github.com/openshift-pipelines/pipeline-service).

## Description

Pipeline Service offers an infrastructure to easily run Tekton Pipelines in a secured and isolated way. Therefore some restrictions need to be set on the workspaces that can consume the Pipeline Service infrastructure.

- Quotas limit the amount of compute resources that can be consumed.
- NetworkPolicies restrict the access granted to the pods running the pipeline tasks.

## Getting Started

### Building the image

Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/settings-operator:tag
```

### Deploying to kcp

The parameter specifying the workspace where the APIExport is located needs to be amended in [the controller deployment](config/manager/manager.yaml) to match the environment.
 
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
make run ARGS="-v 6 --config=config/manager/controller_manager_config_test.yaml --api-export-name=settings-configuration.pipeline-service.io"
```

**NOTE:** You can also run this in one step by running: `make install run`

Here is an example of a launch configuration for VSCode

~~~
{
    // TODO: the kubeconfig path needs to be updated to point to the file
    // in the local environment
    "version": "0.2.0",
    "configurations": [
        {
            "type": "go",
            "request": "launch",
            "name": "Debug App",
            "program": "${workspaceFolder}/main.go",
            "args": [
                "--api-export-name", "settings-configuration.pipeline-service.io"
                "--config", "config/manager/controller_manager_config_test.yaml"
                "-v", "6"
            ],
            "env": {
                "KUBECONFIG":"/tmp/kcp-runtime/.kcp/admin.kubeconfig"
            },
        }

    ]
}
~~~

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
