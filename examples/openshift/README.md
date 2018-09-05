# Running NGINX Ingress Controller on OpenShift

## Prerequisites

* A cluster with OpenShift Release 3.5 and above.
* You must be the cluster administrator to deploy the Ingress controller.
* For NGINX Plus:
    * Build and make available in your cluster the [Ingress controller](../../build) image.
    * Update the container image field in the `nginx-plus-ingress-rc.yaml` file accordingly.


## Steps

1. Avoid conflicts with the OpenShift Router.

    NGINX Plus Ingress controller must be able to bind to ports 80 and 443 of the cluster node, where it is running, like the OpenShift Router. Thus, you need to make sure that the Ingress controller and the Router are running on separate nodes. Additionally, NGINX Plus binds to port 8080 to expose its API and the monitoring dashboard.

    To quickly disable the Router you can run:
    ```
    $ oc scale dc router --replicas=0
    ```

1. Choose a project (namespace). In our example, we choose the default namespace:
    ```
    $ oc project default
    ```

1. Create a service account for the Ingress controller with the name *nginx-ingress*:
    ```
    $ oc create sa nginx-ingress
    ```
1. Create a cluster role for the Ingress controller:
    ```
    $ oc create -f nginx-ingress-role.yaml
    ```
1. Add the created cluster role the Ingress controller service account:
    ```
    $ oc adm policy add-cluster-role-to-user nginx-ingress system:serviceaccount:default:nginx-ingress
    ```
1. Add the privileged SSC to the Ingress controller service account:
    ```
    $ oc adm policy add-scc-to-user privileged  --serviceaccount=nginx-ingress
    ```
1. Create a secret with an SSL certificate and key for the default server of NGINX/NGINX Plus. It is recommended that you use your own certificate and key:
    ```
    $ oc create -f default-server-secret.yaml
    ```
1. Deploy NGINX or NGINX Plus Ingress controller with the service account from the previous step:
    ```
    $ oc create -f nginx-ingress-rc.yaml
    ```
    or
    ```
    $ oc create -f nginx-plus-ingress-rc.yaml
    ```

## Additional Steps for Running the Cafe Application Demo

1. By default, users are not able to work with Ingress resources. To allow that:
    Create a role:
    ```
    $ oc create -f ingress-admin-role.yaml
    ```

    Add this role to the users. As an example, we add this role for the user *developer* from the project *myproject*.
    ```
    $ oc policy add-role-to-user ingress-admin developer -n myproject
    ```
1. The web application from the example is running as the ROOT user, so we need to allow that for the service account *default* from the project *myproject*:
    ```
    $ oc adm policy add-scc-to-user anyuid -z default -n myproject
    ```

1. Now you can login as the user *developer* to the project *myproject* and [deploy the Cafe application](../complete-example#2-deploy-the-cafe-application).
