# NGINX Plus Secret

Get your license.jwt from the F5 trials page //TODO make this sound better

Once you have the license.jwt, create a secret called `license-token`.

This is an Opaque secret where the `license.jwt` field should be the base64 encoded jwt token.

Run the following command

```shell
kubectl create secret generic license-token --from-file=license.jwt
```
