# NGINX Ingress Controller Docs

This directory contains all of the user documentation for NGINX Ingress Controller, as well as the requirements for linting, building, and publishing the documentation.

Docs are written in Markdown. We build the docs using [Hugo](https://gohugo.io) and host them on [Netlify](https://www.netlify.com/).

## Setup

To install Hugo locally, refer to the [Hugo installation instructions](https://gohugo.io/getting-started/installing/).

> **NOTE**: We are currently running [Hugo v0.115.3](https://github.com/gohugoio/hugo/releases/tag/v0.115.3) in production.

## Local Docs Development

To build the docs locally, run the desired `make` command from the docs directory:

```text
make clean          -   removes the local `public` directory, which is the default output path used by Hugo
make docs           -   runs a local hugo server so you can view docs in your browser while you work
make hugo-mod       -   cleans the Hugo module cache and fetches the latest version of the theme module
make hugo-mod-tidy  -   removes unused entries in go.mod and go.sum and verifies the dependencies
make docs-drafts    -   runs the local hugo server and includes all docs marked with `draft: true`
```

## Add new docs

### Generate a new doc file using Hugo

To create a new doc file that contains all of the pre-configured Hugo front-matter and the docs task template, **run the following command in the docs directory**:

`hugo new <SECTIONNAME>/<FILENAME>.<FORMAT>`

For example:

```shell
hugo new getting-started/install.md
```

The default template -- task -- should be used for most docs. To create docs using the other content templates, you can use the `--kind` flag:

```shell
hugo new tutorials/deploy.md --kind tutorial
```

The available content types (`kind`) are:

- concept: Helps a customer learn about a specific feature or feature set.
- tutorial: Walks a customer through an example use case scenario; results in a functional PoC environment.
- reference: Describes an API, command line tool, config options, etc.; should be generated automatically from source code.
- troubleshooting: Helps a customer solve a specific problem.
- openapi: Contains front-matter and shortcode for rendering an openapi.yaml spec

## How to format docs

### How to format internal links

Format links as [Hugo refs](https://gohugo.io/content-management/cross-references/).

- File extensions are optional.
- You can use relative paths or just the filename. (**Paths without a leading / are first resolved relative to the current page, then to the remainder of the site.**)
- Anchors are supported.

For example:

```md
To install <product>, refer to the [installation instructions]({{< ref "install" >}}).
```

### How to include images

You can use the `img` [shortcode](#how-to-use-hugo-shortcodes) to add images into your documentation.

1. Add the image to the static/img directory, or to the same directory as the doc you want to use it in.

   - **DO NOT include a forward slash at the beginning of the file path.** This will break the image when it's rendered.
     See the docs for the [Hugo relURL Function](https://gohugo.io/functions/relurl/#input-begins-with-a-slash) to learn more.

1. Add the img shortcode:

    {{< img src="<img-file.png>" >}}

> Note: The shortcode accepts all of the same parameters as the [Hugo figure shortcode](https://gohugo.io/content-management/shortcodes/#figure).

### How to use Hugo shortcodes

You can use [Hugo shortcodes](/docs/themes/f5-hugo/layouts/shortcodes/) to do things like format callouts, add images, and reuse content across different docs.

For example, to use the note callout:

```md
{{< note >}}Provide the text of the note here. {{< /note >}}
```

The callout shortcodes also support multi-line blocks:

```md
{{< caution >}}
You should probably never do this specific thing in a production environment.

If you do, and things break, don't say we didn't warn you.
{{< /caution >}}
```

Supported callouts:

- `caution`
- `important`
- `note`
- `see-also`
- `tip`
- `warning`

A few more fun shortcodes:

- `fa`: inserts a Font Awesome icon
- `img`: include an image and define things like alt text and dimensions
- `include`: include the content of a file in another file; the included file must be present in the content/includes directory
- `link`: makes it possible to link to a file and prepend the path with the Hugo baseUrl
- `openapi`: loads an OpenAPI spec and renders as HTML using ReDoc
- `raw-html`: makes it possible to include a block of raw HTML
- `readfile`: includes the content of another file in the current file; does not require the included file to be in a specific location.
