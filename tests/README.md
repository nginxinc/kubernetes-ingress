# Tests

The project includes automated tests for testing the Ingress Controller in a Kubernetes cluster. The tests are written in Python3 and use the pytest framework.

Below you will find the instructions on how to run the tests against a Minikube cluster. However, you are not limited to Minikube and can use other types of Kubernetes clusters. See the [Configuring the Tests](#configuring-the-tests) section to find out about various configuration options.

## Running Tests in Minikube

### Prerequisites:

* Minikube. 
* Python3 or Docker.

#### Step 1 - Create a Minikube Cluster

```bash
$ minikube start
```

#### Step 2 - Run the Tests
 
**Note**: if you have the Ingress Controller deployed in the cluster, please uninstall it first, making sure to remove its namespace and RBAC resources.

Run the tests:
* Use local Python3 installation:
    ```bash
    $ cd tests
    $ pip3 install -r requirements.txt
    $ python3 -m pytest --node-ip=$(minikube ip)
    ```
* Use Docker:
    ```bash
    $ cd tests
    $ make build
    $ make run-tests NODE_IP=$(minikube ip)
    ```
The tests will use the Ingress Controller for NGINX with the default *nginx/nginx-ingress:edge* image. See the section below to learn how to configure the tests including the image and the type of NGINX -- NGINX or NGINX Plus.

## Configuring the Tests

The table below shows various configuration options for the tests. If you use Python3 to run the tests, use the command-line arguments. If you use Docker, use the [Makefile](Makefile) variables.


| Command-line Argument | Makefile Variable | Description | Default |
| :----------------------- | :------------ | :------------ | :----------------------- |
| `--context` | `CONTEXT` | The context to use in the kubeconfig file. | `""` |
| `--image` | `BUILD_IMAGE` | The Ingress Controller image. | `nginx/nginx-ingress:edge` |
| `--image-pull-policy` | `PULL_POLICY` | The pull policy of the Ingress Controller image. | `IfNotPresent` |
| `--deployment-type` | `DEPLOYMENT_TYPE` | The type of the IC deployment: deployment or daemon-set. | `deployment` |
| `--ic-type` | `IC_TYPE` | The type of the Ingress Controller: nginx-ingress or nginx-ingress-plus. | `nginx-ingress` |
| `--service` | `SERVICE` | The type of the Ingress Controller service: nodeport or loadbalancer. | `nodeport` |
| `--node-ip` | `NODE_IP` | The public IP of a cluster node. Not required if you use the loadbalancer service (see --service argument). | `""` |
| `--kubeconfig` | `N/A` | An absolute path to a kubeconfig file. | `~/.kube/config` or the value of the `KUBECONFIG` env variable |
| `N/A` | `KUBE_CONFIG_FOLDER` | A path to a folder with a kubeconfig file. | `~/.kube/` |
| `--show-ic-logs` | `SHOW_IC_LOGS` | A flag to control accumulating IC logs in stdout. | `no` |
| `N/A` | `PYTEST_ARGS` | Any additional pytest command-line arguments (i.e `-m "smoke"`) | `""` |

If you would like to use an IDE (such as PyCharm) to run the tests, use the [pytest.ini](pytest.ini) file to set the command-line arguments.

Tests are marked with custom markers. The markers allow to logically split all the tests into smaller groups. The full list can be found in the [pytest.ini](pytest.ini) file or via command line:
```bash
$ python3 -m pytest --markers
```