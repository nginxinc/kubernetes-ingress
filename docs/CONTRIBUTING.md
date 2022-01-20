# CONTRIBUTING

## Overview

Welcome! This document describes the tools and processes for contributing to the NGINX Ingress Controller documentation. 

- All public-facing docs are in the `content` directory. 
- Files that should be included in the public docs but not sent through the HTML processor live in the `static` directory.
- Supporting example files can live in the same directory as the file that calls them. 
  
  > If a file is going to be used in more than one document, place it in the `/static` directory.
- Image files live in `static/img`.

## Tools

We write docs in [Markdown](https://www.markdownguide.org/) and build them using [Hugo](https://gohugo.io/).

> NOTE: The NGINX Ingress Controller docs rely on a Hugo theme that is developed by F5 NGINX as a private module. Unfortunately, non-F5 contributors will run into authentication errors when trying to run Hugo because the theme module is not publicly available. These contributors will be able to access docs previews via Netlify after opening a pull request in GitHub.

## Prerequisites

- [Golang](https://golang.org/doc/install)
- [Hugo](https://gohugo.io/getting-started/installing/)

## Guidelines

### Add a new doc

→ To create a new doc that contains all of the pre-configured Hugo front-matter and the docs task template:

```shell
hugo new <SECTIONNAME>/<FILENAME>.<FORMAT>
```

e.g.,

```shell
hugo new install.md
```

The default template -- task -- should apply to most docs.

→ To create other types of docs, you can add the `--kind` flag:

```shell
hugo new tutorials/deploy.md --kind tutorial
```

The available kinds are:

- **Task**: Enable the end-user to achieve a specific goal, based on use case scenarios.
- **Concept**: Help an end-user learn about a specific feature or feature set.
- **Reference**: Describes an API, command line tool, config options, etc.; should be generated automatically from source code. 
- **Troubleshooting**: Helps an end-user solve a specific problem.
- **Tutorial**: Walk an end-user through an example use case scenario; results in a functional PoC environment.

### How to format internal links

Format links as [Hugo refs](https://gohugo.io/content-management/cross-references/).

- Using file extensions is optional.
- You can use a relative path or just the filename. (Paths without a leading `/` are first resolved relative to the current page, then to the remainder of the site.)
- Anchors are supported.

For example:

```md
To install NGINX Ingress Controller by building your own Docker image, refer to the [installation instructions]({{< ref "installation/building-ingress-controller-image" >}}).
```

### How to insert images

You can use the `img` shortcode to insert images into your documentation.

1. Add the image to the `static/img` directory.
2. Add the img shortcode:

    `{{< img src="<img-file.png>" >}}`

The shortcode accepts all of the same parameters as the [Hugo figure shortcode](https://gohugo.io/content-management/shortcodes/#figure).

### Using Hugo shortcodes

You can use Hugo shortcodes to do things like format callouts, add images, and reuse content across different docs.

For example, to use the `note` callout:

```md
{{< note >}}Provide the text of the note here. {{< /note >}}
```

The callout shortcodes also support multi-line blocks:

```md
{{< caution >}}
You should probably never do this specific thing in a production environment. If you do, and things break, don't say we didn't warn you.
{{< /caution >}}
```

Supported callouts:

- caution
- important
- note
- see-also
- tip
- warning

A few more cool shortcodes:
- bootstrap-table: adds scrollbars and formatting to tables by accepting bootstrap class names
- fa: inserts a Font Awesome icon
- include: include the content of a file in another file (requires the included file to be in the `content/includes` directory)
- link: makes it possible to link to a static file and prepend the path with the Hugo `baseUrl` (helpful for download links)
- openapi: loads an OpenAPI spec and renders as HTML using ReDoc
- raw-html: makes it possible to include a block of raw HTML
- readfile: includes the content of another file in the current file

## Makefile Usage

```txt
make all                \\ clean, get, and vendor all Hugo modules; build docs
all-local               \\ clean up local Hugo build directory (public); 
                        \\   clean, get, and vendor all modules; build docs
clean                   \\ clean up local Hugo build directory (public)
hugo-mod                \\ clean, get, and vendor all Hugo modules
build-production        \\ build docs using production environment settings
build-staging           \\ build docs using staging environment settings
hugo-server-drafts      \\ run local server using production environment settings 
                        \\   and render all docs in draft status
hugo-server             \\ run local server using production environment settings 
```

## Manage Docs Content