##  Configuring a NIC `daemonsets` to listen with non-default different ports

Using a deployment or a daemonset is a personal preference for Ingress. In this tutorial I have assumed that you have already decided on a daemonset and you are in a situation where you want to run multiples. Perhaps you have one that is handling the additional load of WAF and the other isn't.  Perhaps you have a need to isolate some of your tenants.  The reason why is not as important.

Lets jump into this together. You have one daemonset of NGINX Ingress Controller up and running and you need to add a second daemonset. Do to port assignment rules you can't have both deployments listening on the same port and to avoid the port collision.

I am going to walk you through adding the second daemonset.  Through this process I am outlining how to change te default listening port for NGINX Ingress Controller.

Is this example this second daemonset is listening on ports 8080 and 8443.  (the defaults are 80 and 443)

### Using the NIC `Helm` chart

First, pull down the NIC Helm chart as outlined in the installation doc: https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/#getting-the-chart-sources

The first thing you need to do is comment out, these lines (59-64) in the *helm templates* for the daemonset:   
https://github.com/nginxinc/kubernetes-ingress/blob/master/deployments/helm-chart/templates/controller-daemonset.yaml#L59-L64
```
        - name: http
          containerPort: 80
          hostPort: 80
        - name: https
          containerPort: 443
          hostPort: 443
```

Then update `values.yaml` and use the `customPorts` section to add the specific ports you want your NGINX Ingress `daemonset` to listen on.
For example, here is what I tested: (this is what I specified in my helm `values.yaml` file, under `customPorts`)
```
    customPorts:
    - name: http
      containerPort: 8080
      protocol: TCP
    - name: https
      containerPort: 8443
      protocol: TCP
```
I set my `daemonset` to listen on ports 85 and 8443.
You could have a second `helm` deployment, using default 80 or 443, or even something like 85 and 9443.

*NOTE:* Since this example includes a second NGINX Ingress deployment, you need to create a second `ingressClass`, one for each deployment.
Make sure you change `ingressClass` in the `values.yaml` when defining your second deployment.

Then you can use helm to deploy NIC.

<file is modified locally - give me a HELM command to be complete>

After deploying with Helm you should see two NIC deployments.  Both running as a daemonset, with one pod on each node.
In my lab this looked like:

```
k get po,svc -n nginx-ingress -owide
NAME                           READY   STATUS    RESTARTS   AGE   IP           NODE       NOMINATED NODE   READINESS GATES
pod/nic1-nginx-ingress-j2w5b   1/1     Running   0          28s   172.17.0.3   dev01   <none>           <none>

NAME                         TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE   SELECTOR
service/nic1-nginx-ingress   LoadBalancer   10.103.67.173   172.16.24.2     80:32345/TCP,443:30966/TCP   28s   app=nic1-nginx-ingress
```
```
❯ k get pods -A -owide
NAMESPACE        NAME                                      READY   STATUS    RESTARTS   AGE   IP          NODE                 NOMINATED NODE   READINESS GATES
kube-system      metrics-server-86cbb8457f-82ghk           1/1     Running   0          52m   10.42.0.2   dev01-server-0   <none>           <none>
kube-system      local-path-provisioner-5ff76fc89d-zsjdh   1/1     Running   0          52m   10.42.0.3   dev01-server-0   <none>           <none>
kube-system      coredns-7448499f4d-2dwm5                  1/1     Running   0          52m   10.42.0.4   dev01-server-0   <none>           <none>
nginx-ingress    nic1-nginx-ingress-8s2j5                  1/1     Running   0          51m   10.42.0.5   dev01-server-0   <none>           <none>
nginx-ingress    nic1-nginx-ingress-dkkbf                  1/1     Running   0          51m   10.42.1.2   dev01-agent-0    <none>           <none>
nginx-ingress2   nic2-nginx-ingress-2w85g                  1/1     Running   0          12s   10.42.0.8   dev01-server-0   <none>           <none>
nginx-ingress2   nic2-nginx-ingress-7x6zq                  1/1     Running   0          12s   10.42.1.5   dev01-agent-0    <none>           <none>
```
- Describe on the daemonset
```
Controlled By:  DaemonSet/nginx-ingress-nginx-ingress
Containers:
  nginx-ingress-nginx-ingress:
    Container ID:  containerd://547bb21d418ad01c8f8adff67e62d831fd0f615e52d336aa07fee61c0aa1bbc8
    Image:         nginx/nginx-ingress:2.1.0
    Image ID:      docker.io/nginx/nginx-ingress@sha256:b94a2ff0fbd36fa7c2b130226475e0cf924dd6a7e038cd20be74149875e3c18a
    Ports:         8080/TCP, 8443/TCP, 9113/TCP, 8081/TCP
    Host Ports:    0/TCP, 0/TCP, 0/TCP, 0/TCP
```
As you can see in the above output, is my **second** deployment of NGINX Ingress Controller as a Daemonset. It is now running on ports 8080 and 8443, different from my first deployment, which was running on ports 80 and 443. 

Now I have two NGINX Ingress Controller daemonset deployments on the same cluster.

See the `values.yaml` file in this repo as a reference to what was used.

<this example should be provided>
<note I updated port 85 to 8080 and the start of the document described>
