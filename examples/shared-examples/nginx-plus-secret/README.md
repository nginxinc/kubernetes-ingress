# NGINX Plus Secret

[Download your NGINX Plus license file](https://docs.nginx.com/nginx-ingress-controller/installation/nic-images/get-image-using-jwt/#before-you-begin]) and save it to a file called `license.jwt`.

Once you have the license.jwt, create a secret called `license-token`.

This is a secret of type `nginx.com/license` where the `license.jwt` field should be the base64 encoded jwt token.

Run the following commands

```shell
kubectl create namespace nginx-ingress
kubectl create secret generic license-token --from-file=license.jwt=<path-to-your-license-file> --type=nginx.com/license -n nginx-ingress
```
