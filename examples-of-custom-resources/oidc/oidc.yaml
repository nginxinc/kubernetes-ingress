apiVersion: k8s.nginx.org/v1
kind: Policy
metadata:
  name: oidc-policy
spec:
  oidc:
    clientID: nginx-plus
    clientSecret: oidc-secret
    authEndpoint: https://keycloak.example.com/auth/realms/master/protocol/openid-connect/auth
    tokenEndpoint: http://keycloak.default.svc.cluster.local:8080/auth/realms/master/protocol/openid-connect/token
    jwksURI: http://keycloak.default.svc.cluster.local:8080/auth/realms/master/protocol/openid-connect/certs
    scope: openid+profile+email
