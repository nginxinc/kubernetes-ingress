---
title: "{{ replace .Name "-" " " | title }}"
date: {{ .Date }}
# Change draft status to false to publish doc
draft: true
# Description
# Add a short description (150 chars) for the doc. Include keywords for SEO. 
# The description text appears in search results and at the top of the doc.
description: ""
# Assign weights in increments of 100
weight: 
toc: true
tags: [ "docs" ]
# Create a new entry in the Jira DOCS Catalog and add the ticket ID (DOCS-<number>) below
docs: "DOCS-000"
# Taxonomies
# These are pre-populated with all available terms for your convenience.
# Remove all terms that do not apply.
categories: ["installation", "platform management", "load balancing", "api management", "service mesh", "security", "analytics"]
doctypes: ["troubleshooting"]
journeys: ["researching", "getting started", "using", "renewing", "self service"]
personas: ["devops", "netops", "secops", "support"]
versions: []
authors: []
---

## Overview

Briefly describe the goal of this document, that is, what the user will accomplish by reading what follows.

## Issue 1 - write as a verb phrase

Explain the issue. Include any identifying details, such as error messages.

When the system does xyz, you may see an error similar to the following:

```text
error message here
```

This issue is caused by -- add cause here.

To resolve the issue, take the following steps:

1. The steps
2. To take to
3. Resolve the issue.

## Issue 2 - write as a verb phrase

## Issue 3 - write as a verb phrase