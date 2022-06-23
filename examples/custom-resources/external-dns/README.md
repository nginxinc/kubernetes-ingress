Step 1: Register the external-crd with the k8s api (run from the root of this repo):

    ```k apply -f deployments/common/crds/externaldns.nginx.org_dnsendpoints.yaml```

Step 2: Deploy external-dns

    Update `external-dns-route53.yaml` with your Domain Name and Hosted Zone ID, and apply the file.

    ```k apply -f external-dns-route53.yaml```

Step 3: Deploy the DNSEndpoint object

    Update `dnsendpoint.yaml` with the DNS hostname and the target IPs (the external IPs of the Ingress Controller service), and apply the file.

    ```k apply -f dnsendpoint.yaml```

Step 4: Check the logs of the external-dns pod, and you'll see something like this:

```
time="2022-05-26T15:04:45Z" level=info msg="Desired change: CREATE cafe.example.com A [Id: /hostedzone/Z04ABCDEFGHIJKLMNO]"
time="2022-05-26T15:04:45Z" level=info msg="Desired change: CREATE cafe.example.com TXT [Id: /hostedzone/Z04ABCDEFGHIJKLMNO]"
time="2022-05-26T15:04:46Z" level=info msg="2 record(s) in zone example.com. [Id: /hostedzone/Z04ABCDEFGHIJKLMNO] were successfully updated"
time="2022-05-26T15:05:45Z" level=info msg="Applying provider record filter for domains: [example.com. .example.com.]"
time="2022-05-26T15:05:45Z" level=info msg="Desired change: UPSERT cafe.example.com A [Id: /hostedzone/Z04ABCDEFGHIJKLMNO]"
time="2022-05-26T15:05:45Z" level=info msg="Desired change: UPSERT cafe.example.com TXT [Id: /hostedzone/Z04ABCDEFGHIJKLMNO]"
time="2022-05-26T15:05:46Z" level=info msg="2 record(s) in zone example.com. [Id: /hostedzone/Z04ABCDEFGHIJKLMNO] were successfully updated"
time="2022-05-26T15:06:45Z" level=info msg="Applying provider record filter for domains: [example.com. .example.com.]"
time="2022-05-26T15:06:45Z" level=info msg="Desired change: UPSERT cafe.example.com TXT [Id: /hostedzone/Z04ABCDEFGHIJKLMNO]"
time="2022-05-26T15:06:45Z" level=info msg="Desired change: DELETE cafe.example.com A [Id: /hostedzone/Z04ABCDEFGHIJKLMNO]"
time="2022-05-26T15:06:45Z" level=info msg="Desired change: CREATE cafe.example.com A [Id: /hostedzone/Z04ABCDEFGHIJKLMNO]"
time="2022-05-26T15:06:46Z" level=info msg="3 record(s) in zone example.com. [Id: /hostedzone/Z04ABCDEFGHIJKLMNO] were successfully updated"
```
