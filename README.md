# Angi Controller Practical
A simple K8s Operator for deploying PodInfo with a backing Redis Cache

## Description
The Operator watches the MyAppResource CR, which looks like:
```
apiVersion: my.api.group/v1alpha1
kind: MyAppResource
metadata:
  name: whatever
spec:
  replicaCount: 2
  # i changed the spec here to match k8s conventions
  resources:
    requests:
      cpu: 100m
    limits:
      memory: 64Mi
  image:
    repository: ghcr.io/stefanprodan/podinfo
    tag: latest
  ui:
    color: "#34577c"
    message: "some string"
  redis:
    enabled: true
```

And maps those settings into fields within [PodInfo](https://github.com/stefanprodan/podinfo) and [Redis](https://github.com/stefanprodan/podinfo) Deployments.

The source code was scaffolded with kubebuilder, see below for the many `make` commands. But the simplest getting started is this:

1. Run UTs:
```
make test
```

2. Install CRDs:
```
make install
```

3. Run the operator locally
```
make run
```

4. Deploy sample CR:
```
kubectl apply -f config/samples/my_v1alpha1_myappresource.yaml
```
*Edit that file and rerun apply to see updates*

5. Connect to Pod Info Endpoint with port-forward
```
kubectl port-forward deployment/whatever 9898
curl http://localhost:9898
```

6. Delete CR:
```
kubectl delete -f config/samples/my_v1alpha1_myappresource.yaml
```

7. Stop the operator with `Ctrl+C`

8. Uninstall CRDs:
```
make uninstall
```


## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/angi:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/angi:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

