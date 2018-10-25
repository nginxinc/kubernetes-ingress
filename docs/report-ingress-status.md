# Reporting Status of Ingress Resources

An Ingress resource can have a status that includes the address (an IP address or a DNS name), through which the hosts of that Ingress resource are publicly accessible.
You can see the address in the output of the `kubectl get ingress` command, in the ADDRESS column, as shown below:

```
$ kubectl get ingresses
NAME           HOSTS              ADDRESS           PORTS     AGE
cafe-ingress   cafe.example.com   12.13.23.123      80, 443   2m
```

The Ingress controller must be configured to report an Ingress status:

1. Use the command-line flag `-report-ingress-status`.
2. Define a source for an external address. This can be either of:
    1. A user defined address, specified in the `external-status-address` [ConfigMap key](configmap-and-annotations.md).
    2. A Service of the type LoadBalancer configured with an external IP or address and specified by the `-external-service` command-line flag.
3. If you're running multiple replicas of the Ingress controller, enable leader election with the `-enable-leader-election` flag
to ensure that only one replica updates an Ingress status.

Notes: The Ingress controller does not clear the status of Ingress resources when it is being shut down.
