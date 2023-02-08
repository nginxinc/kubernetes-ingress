---
title: "{{ replace .Name "-" " " | title }}"
date: {{ .Date }}
# Change draft status to false to publish doc
draft: true
# Description
# Add a short description (150 chars) for the doc. Include keywords for SEO. 
# The description text appears in search results and at the top of the doc.
description: "Add a one-sentence description of the doc that'll show up in search results."
# Assign weights in increments of 100
weight: 
toc: true
tags: [ "docs" ]
# Create a new entry in the Jira DOCS Catalog and add the ticket ID (DOCS-<number>) below
docs: "DOCS-000"
# Taxonomies
# These are pre-populated with all available terms for your convenience.
# Remove all terms that do not apply.
categories: ["installation", "platform management", "load balancing", "api management", "security", "analytics"]
doctypes: ["reference"]
journeys: ["researching", "getting started", "using", "renewing", "self service"]
personas: ["devops", "netops", "secops", "support"]
versions: []
authors: []
---

{{<custom-styles>}}

## Overview

Write an introduction for the policy. Briefly explain what the policy is for.

Introduce and explain any new concepts the user may need to understand before proceeding.

---

## Before You Begin

To complete the steps in this guide, you need the following:

- API Connectivity Manager is installed, licensed, and running
- You have [one or more Environments with an API Gateway]({{< relref "acm/getting-started/add-api-gateway.md" >}})
- You have [published one or more API Gateways]({{< relref "acm/getting-started/publish-api-proxy.md" >}})

### How to Access the User Interface

{{< include "acm/how-to/access-acm-ui" >}}

### How to Access the REST API

{{< include "acm/how-to/access-acm-api" >}}

---

## Create an XYZ Policy

{{<tabs name="policy-implementation">}}

{{%tab name="UI"%}}

To create an XYZ policy using the web interface:

1. Go to the FQDN for your NGINX Management Suite host in a web browser and log in. Then, from the Launchpad menu, select **API Connectivity Manager**.
2. Add other steps here
3. As a numbered list.

{{%/tab%}}

{{%tab name="API"%}}

To create an XYZ policy using the REST API, send an HTTP `POST` request to the Add-Endpoint-Name-Here endpoint.

{{< raw-html>}}<div class="table-responsive">{{</raw-html>}}
{{< bootstrap-table "table table-striped table-bordered" >}}
| Method | Endpoint            |
|--------|---------------------|
| `POST` | `/path/to/endpoint` |
{{</bootstrap-table>}}
{{< raw-html>}}</div>{{</raw-html>}}

<details open>
<summary>JSON request</summary>

``` json
{
  "users": [
    {
      "id": 1,
      "name": "John Doe",
      "age": 24
    },
    {
      "id": 2,
      "name": "Jane Doe",
      "age": 28
    }
  ]
}
```

</details>

<br>

{{< raw-html>}}<div class="table-responsive">{{</raw-html>}}
{{< bootstrap-table "table table-striped table-bordered" >}}
| Field  | Datatype | Possible Values     | Description                                        | Required | Default               |
|--------|----------|---------------------|----------------------------------------------------|----------|-----------------------|
| `id`   | integer  | A unique int >= 1   | Description for value.                             | Yes      | System assigned       |
| `name` | string   | Example: `Jane Doe` | A short description of what the field is used for. | Yes      | Add the default value |
| `age`  | integer  | 1–110               | Description for the value                          | Yes      |                       |

{{< /bootstrap-table >}}
{{< raw-html>}}</div>{{</raw-html>}}

{{%/tab%}}

{{</tabs>}}

---

## Verify the Policy

Confirm that the policy has been set up and configured correctly by taking these steps:

- Add verification steps here

---

## Troubleshooting

For help resolving common issues when setting up and configuring the policy, follow the steps in this section. If you cannot find a solution to your specific issue, reach out to [NGINX Customer Support]({{< relref "support/contact-support.md" >}}) for assistance.

### Issue 1

Add a description for the issue. Include any error messages users might see.

Resolution/workaround:

- Add steps here.

### Issue 2

Add a description for the issue. Include any error messages users might see.

Resolution/workaround:

- Add steps here.