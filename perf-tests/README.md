# AP Performance (reload and response times) Tests

The project includes automated performance tests for Ingress Controller with AppProtect module in a Kubernetes cluster. The tests are written in Python3 and use the pytest framework for reload tests and locust.io for API tests.

Below you will find the instructions on how to run the tests against a Minikube cluster. However, you are not limited to Minikube and can use other types of Kubernetes clusters. See the [Configuring the Tests](#configuring-the-tests) section to find out about various configuration options.

## Running Tests in Minikube

### Prerequisites:

* Minikube.
* Python3 (in a virtualenv)

#### Step 1 - Create a Minikube Cluster

```bash
$ minikube start
```

#### Step 2 - Run the Performance Tests

**Note**: if you have the Ingress Controller deployed in the cluster, please uninstall it first, making sure to remove its namespace and RBAC resources.

Run the tests:

* Use local Python3 installation (advised to use pyenv/virtualenv):
    ```bash
    $ cd perf_tests
    $ pip3 install -r requirements.txt
    $ pytest -v -s -m ap_perf --count=<INT> --node-ip=$(minikube ip) --users=<INT> --hatch-rate=<INT> --time=<INT>
    ```

The tests will use the Ingress Controller for NGINX with the image built from `debian-image-nap-plus`. See the section below to learn how to configure the tests including the image and the type of NGINX -- NGINX or NGINX Plus.
Refer the [Configuring the Tests](#configuring-the-tests) section for valid arguments.

## Configuring the Tests

The table below shows various configuration options for the performance tests. Use command line arguments to run tests with Python3

| Command-line Argument | Description | Default |
| :----------------------- | :------------ | :----------------------- |
| `--context` | The context to use in the kubeconfig file. | `""` |
| `--image` | The Ingress Controller image. | `nginx/nginx-ingress:edge` |
| `--image-pull-policy` | The pull policy of the Ingress Controller image. | `IfNotPresent` |
| `--deployment-type` | The type of the IC deployment: deployment or daemon-set. | `deployment` |
| `--ic-type` | The type of the Ingress Controller: nginx-ingress or nginx-ingress-plus. | `nginx-ingress` |
| `--service` | The type of the Ingress Controller service: nodeport or loadbalancer. | `nodeport` |
| `--node-ip` | The public IP of a cluster node. Not required if you use the loadbalancer service (see --service argument). | `""` |
| `--kubeconfig` | An absolute path to a kubeconfig file. | `~/.kube/config` or the value of the `KUBECONFIG` env variable |
| `N/A` | A path to a folder with a kubeconfig file. | `~/.kube/` |
| `--show-ic-logs` | A flag to control accumulating IC logs in stdout. | `no` |
| `N/A` | Any additional pytest command-line arguments (i.e `-m "smoke"`) | `""` |
| `--count` | Number of times to repeat tests | `1` |
| `--users` | Total no. of users/locusts for response perf tests. | `10` |
| `--hatch-rate` | No. of users hatched per second. | `5` |
| `--time` | Duration for AP response perf tests in seconds. | `10` |
