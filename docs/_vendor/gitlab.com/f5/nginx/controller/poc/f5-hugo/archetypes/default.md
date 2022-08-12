---
title: "{{ replace .Name "-" " " | title }}"
date: {{ .Date }}
# Change draft status to false to publish doc.
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
doctypes: ["task"]
journeys: ["researching", "getting started", "using", "renewing", "self service"]
personas: ["devops", "netops", "secops", "support"]
versions: []
authors: []

---

## Overview

Briefly describe the goal of this document, that is, what the user will learn or accomplish by reading what follows.

Introduce and explain any new concepts the user may need to understand before proceeding.

## Before You Begin

To complete the instructions in this guide, you need the following:

1. Provide any prerequisites here.
2. Format as a numbered or bulleted list as appropriate.
3. Keep the list entries grammatically parallel.1. Provide any prerequisites here.

## Goal 1 - write as a verb phrase

Add introductory text. Say what the user will be doing.

To do xzy, take the following steps:

1. This is where you provide the steps that the user must take to accomplish the goal.

    ```bash
    code examples should be nested within the list
    ```

2. Format as numbered lists.

    {{< note >}}Add notes like this.{{</note>}}

3. If there is only one step, you don't need to format it as a numbered list.

## Goal 2 - write as a verb phrase

## Goal 3  - write as a verb phrase

## Discussion

Use the discussion section to expand on the information presented in the steps above.

This section contains the "why" information.

This information lives at the end of the document so that users who just want to follow the steps don't have to scroll through a wall of explanatory text to find them.

## Verification

Explain how the user can verify the steps completed successfully.

## What's Next

- Provide up to 5 links to related topics (optional).
- Format as a bulleted list.
