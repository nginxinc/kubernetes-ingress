---
title: "{{ replace .Name "-" " " | title }}"
date: {{ .Date }}
draft: true
description: ""
# Assign weights in increments of 100
weight: 
draft: false
doctypes: ["reference"]
toc: true
tags: [ "api" ]
menu: api
layout: api
# Taxonomies
# These are pre-populated with all available terms for your convenience.
# Remove all terms that do not apply.
categories: ["installation", "platform management", "load balancing", "api management", "service mesh", "security", "analytics"]
doctypes: ["reference"]
journeys: ["researching", "getting started", "using"]
personas: ["devops", "netops", "secops", "support"]
versions: ["<version>"]
authors: []
---

{{< openapi spec="/path/to/openapi.yaml" >}}
