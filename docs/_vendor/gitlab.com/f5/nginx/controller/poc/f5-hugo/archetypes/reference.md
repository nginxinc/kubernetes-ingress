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
doctypes: ["reference"]
journeys: ["researching", "getting started", "using", "renewing", "self service"]
personas: ["devops", "netops", "secops", "support"]
versions: []
authors: []
---

## Overview

Briefly describe the goal of this document, that is, what the user will learn or accomplish by reading what follows.

Introduce and explain any new concepts the user may need to understand before proceeding.
## Reference 1 - format as a noun phrase

Provide reference material here. This may be tables, paragraphs, imported code, etc.

## Reference 2 - format as a noun phrase

## Reference 3 - format as a noun phrase

## What's Next

- Provide up to 5 links to related topics (optional).
- Format as a bulleted list.
