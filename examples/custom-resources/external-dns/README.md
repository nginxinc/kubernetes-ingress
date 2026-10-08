# External DNS

In this example we configure a VirtualServer resource to integrate with
[ExternalDNS](https://github.com/kubernetes-sigs/external-dns) to make the resource discoverable via a public DNS
server. In this example, we deploy an ExternalDNS deployment with the AWS provider enabled.

## Prerequisites

1. Run `make secrets` command to generate the necessary secrets for the example.
1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/install/manifests)
   instructions to deploy the Ingress Controller with custom resources enabled. Additionally, the Ingress Controller
   must be configured to report the VirtualServer status by setting either the `external-service` command line argument,
   or setting the `external-status-address` key in the ConfigMap resource (see the [Reporting Resources Status
   docs](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/reporting-resources-status#virtualserver-and-virtualserverroute-resources)
   for more details).

### Selecting the DNSEndpoint API group

NGINX Ingress Controller can write `DNSEndpoint` resources to one of two API groups:

- `externaldns.nginx.org/v1` (default) — NIC's own group, kept as the default for backwards compatibility. The CRD is shipped by the NIC Helm chart / manifests.
- `externaldns.k8s.io/v1alpha1` — the upstream external-dns group. The CRD is **not** shipped by NIC to avoid ownership conflicts with the external-dns Helm chart; install it separately from the [external-dns repo](https://github.com/kubernetes-sigs/external-dns).

Which group you need depends on your external-dns version:

- external-dns **v0.20.x and earlier** reads **either** group. It builds its CRD client from whatever you pass to `--crd-source-apiversion`, so it works equally well with `externaldns.nginx.org/v1` and `externaldns.k8s.io/v1alpha1`.
- external-dns **v0.21.0 and newer** reads **only** `externaldns.k8s.io/v1alpha1`. The group is hard-coded, and `--crd-source-apiversion` no longer selects it.

Because v0.20.x accepts both, you can move to the upstream group before upgrading external-dns. See
[Migrating an existing deployment](#migrating-an-existing-deployment) below.

Toggle groups with the `-external-dns-group-version` command-line flag (or `controller.externalDNSGroupVersion` Helm value). This example uses `externaldns.k8s.io/v1alpha1` and pins external-dns to v0.21.0.

Start NIC with:

```shell
-external-dns-group-version=externaldns.k8s.io/v1alpha1
```

### Migrating an existing deployment

If you already run NIC with ExternalDNS enabled on `externaldns.nginx.org/v1`, you can move to the upstream group
without dropping a DNS record. NIC never deletes `DNSEndpoint` resources, and it does not convert existing ones, so the
cut-over is a sequence you drive yourself.

Migrate **before** upgrading external-dns to v0.21.0+, while it still accepts either group:

| Step | Action | Effect on DNS |
| ---- | ------ | ------------- |
| 1 | Install the upstream CRD (see [Step 1](#step-1-deploy-external-dns) below). | None. |
| 2 | Set `-external-dns-group-version=externaldns.k8s.io/v1alpha1` on NIC and restart it. | NIC creates a new `externaldns.k8s.io/v1alpha1` `DNSEndpoint`. The old `externaldns.nginx.org/v1` object stays in place and is still the one external-dns reads. Record untouched. |
| 3 | Verify the new object exists and has been read (see below). | None. |
| 4 | Set `--crd-source-apiversion=externaldns.k8s.io/v1alpha1` on your existing external-dns and restart it. | external-dns now reads the new object. Same hostname, targets, and TTL, so it logs `All records are already up to date`. |
| 5 | Delete the old `externaldns.nginx.org/v1` object. | None — nothing reads it any more. |
| 6 | Upgrade external-dns to v0.21.0 or newer. | None. |

Steps 4 and 5 cause no record churn because both objects share a name and namespace. external-dns derives its ownership
label from those (`crd/<namespace>/<name>`), so the TXT registry record is identical before and after the switch and the
reconciliation plan comes out empty.

To confirm external-dns has read the new object before step 4, compare its observed generation to its generation:

```console
kubectl get dnsendpoints.externaldns.k8s.io <name> -o jsonpath='{.status.observedGeneration}/{.metadata.generation}'
```

Matching values mean external-dns has picked the resource up. external-dns stamps `status.observedGeneration` when it
reads the resource, which happens before it pushes records to your DNS provider — so treat it as confirmation that the
resource was seen, not that the record is already live.

Two things to be careful about:

- **Do not reverse steps 2 and 4.** Pointing external-dns at the upstream group before NIC has created the object leaves
  it with no source for that hostname. Under `--policy=sync` the record is deleted; under `--policy=upsert-only` it
  survives.
- **Keep the window between steps 2 and 4 short.** NIC stops updating the old object as soon as you flip the toggle, but
  external-dns is still reading it. Changes to the VirtualServer in that window are not published.

If you skip step 5, the old object is not left behind permanently: NIC sets an owner reference to the VirtualServer, so
Kubernetes garbage-collects it when the VirtualServer is deleted.

## Step 1: Deploy external-dns

Install the upstream DNSEndpoint CRD (needed because NIC does not ship it):

```console
kubectl apply --server-side=true -f https://raw.githubusercontent.com/kubernetes-sigs/external-dns/v0.21.0/config/crd/standard/dnsendpoints.externaldns.k8s.io.yaml
```

Update `external-dns-route53.yaml` with your Domain Name and Hosted Zone ID, and apply the file.

```console
kubectl apply -f external-dns-route53.yaml
```

## Step 2 - Deploy the Cafe Application

Create the coffee and the tea deployments and services:

```console
kubectl create -f cafe.yaml
```

## Step 3 - Configure Load Balancing and TLS Termination

1. Create the secret with the TLS certificate and key:

    ```console
    kubectl create -f cafe-secret.yaml
    ```

2. Update the `spec.host` field in the `cafe-virtual-server.yaml` to correspond to your Domain Name and create the
   VirtualServer resource:

    ```console
    kubectl create -f cafe-virtual-server.yaml
    ```

## Step 4 - Test the Configuration

Using a browser, navigate to `https://cafe.<YOUR_DOMAIN_NAME>/coffee`, making sure to update <YOUR_DOMAIN_NAME> as
listed in the `spec.host` of the virtual server. You should see something like the following in the browser window:

```text
Server address: 192.168.86.30:8080
Server name: coffee-6f4b79b975-l484q
Date: 28/Jun/2022:16:01:26 +0000
URI: /coffee
Request ID: 9af5fd7329495819bfb6c6c0f3686a64
```

## Cleaning up DNS records

The external-dns deployment in this example runs with `--policy=upsert-only`, which strips deletions from every
reconciliation plan. With that policy external-dns only ever creates and updates records, so **deleting a `DNSEndpoint`
resource — or the VirtualServer that owns it — does not remove the DNS record**, nor its `externaldns-` TXT ownership
record. Both remain at your DNS provider.

This is expected external-dns behaviour, not a resource leak. The `DNSEndpoint` resource itself does go away —
Kubernetes garbage-collects it with the VirtualServer that owns it — but removing the DNS record is external-dns's
decision, and `upsert-only` forbids it.

To have records removed when the resources that created them go away, run external-dns with `--policy=sync`. Otherwise,
delete leftover records at your DNS provider yourself.
