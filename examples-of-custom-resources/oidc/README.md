# OIDC

In this example, we deploy keycloak and a web application configure load balancing for it via a VirtualServer, and apply an OpenID Connect policy.

## Prerequisites

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) instructions to deploy the Ingress Controller.
1. Save the public IP address of the Ingress Controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
1. Save the HTTP port of the Ingress Controller into a shell variable:
    ```
    $ IC_HTTPS_PORT=<port number>
    ```

## Step 1 - Deploy a Web Application

Create the application deployment and service:
```
$ kubectl apply -f webapp.yaml
```

## Step 2 - Deploy Keycloak

Create keycloak deployment and service:
```
$ kubectl apply -f keycloak.yaml
```

To set up Keycloak, you can either follow the steps in the "Configuring Keycloak" section of the documentation [here](https://docs.nginx.com/nginx/deployment-guides/single-sign-on/keycloak/#configuring-keycloak) or execute the commands [here](./keycloak_setup.md).


## Step 3 - Deploy the Client Secret

1. Edit `client-secret.yaml` with your secret.

1. Create a secret with the name `oidc-secret` that will be used for OIDC validation:
```
$ kubectl apply -f client-secret.yaml
```

## Step 4 - Deploy the OIDC Policy

1. Modify the URL `authEndpoint` in `oidc.yaml` with the public IP address or DNS of keycloak.

1. Create a policy with the name `oidc-policy` that references the secret from the previous step:
```
$ kubectl apply -f oidc.yaml
```

## Step 5 - Deploy the Service for the Ingress Controller and update ConfigMap
1. Deploy the service.
    ```
    $ kubectl apply -f service/nodeport.yaml
    ```
1. Update the ConfigMap with the config required for OIDC.
    ```
    data:
        stream-snippets: |
            resolver 10.96.0.10 valid=5s;
            server {
                listen 12345;
                zone_sync;
                zone_sync_server nginx-ic-svc.nginx-ingress.svc.cluster.local:12345 resolve;
            }
    ```
1. Apply the ConfigMap.
   ```
   $ kubectl apply -f common/nginx-config.yaml
   ```

## Step 6 - Configure Load Balancing and TLS Termination
1. Create the secret with the TLS certificate and key:
    ```
    $ kubectl create -f tls-secret.yaml
    ```

2. Create a VirtualServer resource for the web application:
    ```
    $ kubectl apply -f virtual-server.yaml
    ```

Note that the VirtualServer references the policy `oidc-policy` created in Step 4.

## Step 5 - Test the Configuration

Open a web browser and navigate to the URL of the Ingress Controller
