---
title: Troubleshooting Ingress Resources
description: ""
weight: 300
doctypes: [""]
toc: true
---

Checking the Events of an Ingress Resource
After you create or update an Ingress resource, you can immediately check if the NGINX configuration for that Ingress resource was successfully applied by NGINX:

`kubectl describe ing cafe-ingress`
Name:             cafe-ingress
Namespace:        default
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  12s   nginx-ingress-controller  Configuration for default/cafe-ingress was added or updated
Note that in the events section, we have a Normal event with the AddedOrUpdated reason, which informs us that the configuration was successfully applied.
