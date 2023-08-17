# Keycloak Setup

This guide will help you configure KeyCloak using Keycloak's API:

- Create a `client` with the name `nginx-plus`.
- Add a user `nginx-user` with the password `test`.

**Notes**:

- if you changed the username and password for Keycloak in `keycloak.yaml`, modify the commands accordingly.
- The instructions use [`jq`](https://stedolan.github.io/jq/).

Steps:

1. Save the address of Keycloak into a shell variable:

    ```console
    KEYCLOAK_ADDRESS=keycloak.example.com
    ```

1. Retrieve the access token and store it into a shell variable:

    ```console
    TOKEN=`curl -sS -k --data "username=admin&password=admin&grant_type=password&client_id=admin-cli" https://${KEYCLOAK_ADDRESS}/auth/realms/master/protocol/openid-connect/token | jq -r .access_token`
    ```

    ***Note***: The access token lifespan is very short. If it expires between commands, retrieve it again with the
    command above.
1. Create the user `nginx-user`:

    ```console
    curl -sS -k -X POST -d '{ "username": "nginx-user", "enabled": true, "credentials":[{"type": "password", "value": "test", "temporary": false}]}' -H "Content-Type:application/json" -H "Authorization: bearer ${TOKEN}" https://${KEYCLOAK_ADDRESS}/auth/admin/realms/master/users
    ```

1. Create the client `nginx-plus` and retrieve the secret:

    ```console
    SECRET=`curl -sS -k -X POST -d '{ "clientId": "nginx-plus", "redirectUris": ["https://webapp.example.com:443/_codexch"] }' -H "Content-Type:application/json" -H "Authorization: bearer ${TOKEN}" https://${KEYCLOAK_ADDRESS}/auth/realms/master/clients-registrations/default | jq -r .secret`
    ```

    If everything went well you should have the secret stored in $SECRET. To double check run:

    ```console
    echo $SECRET
    ```
