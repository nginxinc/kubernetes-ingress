# Keycloak Setup

This guide will help you create a `client` with name `nginx-plus`, and a user `nginx-user` with password `test` using Keycloak's API.

If you changed username and password for keycloak in `keycloak.yaml`, modify the commands accordingly.

1. Save the address of Keycloak into a shell variable:
    ```console
    $ KEYCLOAK_ADDRESS=keycloak.example.com:$IC_HTTPS_PORT
    ```
1. Retrieve the access token and store into a shell variable:
    ```console
    $ TOKEN=`curl -sS -k --data "username=admin&password=admin&grant_type=password&client_id=admin-cli" https://${KEYCLOAK_ADDRESS}/auth/realms/master/protocol/openid-connect/token | jq -r .access_token`
    ```
***Note***: The access token lifespan is very short. If it expires between commands, retrieve it again with the command above.
1. Create the user `nginx-user`
    ```console
    $ curl -sS -k -X POST -d '{ "username": "nginx-user", "enabled": true, "credentials":[{"type": "password", "value": "test", "temporary": false}]}' -H "Content-Type:application/json" -H "Authorization: bearer ${TOKEN}" https://${KEYCLOAK_ADDRESS}/auth/admin/realms/master/users
    ```
1. Create the Client `nginx-plus` and retrieve the secret:
    ```console
    $ SECRET=`curl -sS -k -X POST -d '{ "clientId": "nginx-plus", "redirectUris": ["https://webapp.example.com:443/_codexch"] }' -H "Content-Type:application/json" -H "Authorization: bearer ${TOKEN}" https://${KEYCLOAK_ADDRESS}/auth/realms/master/clients-registrations/default | jq -r .secret`
    ```
    If everything went well you should have the secret stored in $SECRET, to double check run:
    ```console
    $ echo $SECRET
    ```
    Now we can encode the secret and copy/paste it in the field `client-secret` inside `client-secret.yaml`:
    ```console
    $ echo -n $SECRET | base64
    ```
