---
title: "{{ replace .Name "-" " " | title }}"
date: {{ .Date }}
draft: true
description: ""
# Assign weights in increments of 100
weight: 
draft: false
toc: true
tags: [ "docs" ]
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
 
Provide a brief introduction to the subject matter, in the context of the product. You do not have to provide background information on the subject in general.
 
For example, if you're writing about how the controller uses RBAC to authenticate users, you don't have to explain the concept of RBAC.
 
## Before You Begin
 
1. Provide any prerequisites here.
2. Format as a numbered or bulleted list as appropriate.
 
## Goal 1
 
These sections contain the "what" and "how" information.
 
The header for each section should tell the user **what** they will accomplish by completing the steps in that section.
 
Provide steps that tell the user **how** to complete the goal.
 
1. This is where you provide the steps that the user must take to accomplish the goal.
 
    ```bash
    code examples should be nested within the list
    ```
 
2. Format as numbered lists.
3. If there is only one step, you don't need to format it as a numbered list.
 
### Goal 1.a
 
Use sub-sections as needed to organize content into easily scannable chunks.
 
## Goal 2
 
## Goal 3
 
## Discussion
 
Use the discussion section to expand on the information presented in the steps above.
 
This section contains the "why" information.
 
This information lives at the end of the document so that users who just want to follow the steps don't have to scroll through a wall of explanatory text to find them.
 
## What's Next
 
- Provide up to 5 links to related topics (optional).
- Format as a bulleted list.
  