---
title: Troubleshooting Policy Resources
description: "."
weight: 200
doctypes: [""]
toc: true
---

Checking the Events of a Policy Resource
After you create or update a Policy resource, you can use `kubectl describe` to check whether or not the Ingress Controller accepted the Policy:

`kubectl describe pol webapp-policy`
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  11s   nginx-ingress-controller  Policy default/webapp-policy was added or updated
Note that in the events section, we have a Normal event with the AddedOrUpdated reason, which informs us that the policy was successfully accepted.

However, the fact that a policy was accepted doesn’t guarantee that the NGINX configuration was successfully applied. To confirm that, check the events of the VirtualServer and VirtualServerRoute resources that reference that policy.

Checking the Events of the ConfigMap Resource
After you update the ConfigMap resource, you can immediately check if the configuration was successfully applied by NGINX:

`kubectl describe configmap nginx-config -n nginx-ingress`
Name:         nginx-config
Namespace:    nginx-ingress
Labels:       <none>
. . .
Events:
  Type    Reason   Age                From                      Message
  ----    ------   ----               ----                      -------
  Normal  Updated  11s (x2 over 26m)  nginx-ingress-controller  Configuration from nginx-ingress/nginx-config was updated
Note that in the events section, we have a Normal event with the Updated reason, which informs us that the configuration was successfully applied.
