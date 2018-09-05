# Contributing Guidelines

The following is a set of guidelines for contributing to the NGINX Ingress Controller. We really appreciate that you are considering contributing!

#### Table Of Contents

[Ask a Question](#ask-a-question)

[Getting Started](#getting-started)

[Contributing](#contributing)

[Style Guides](#style-guides)
  * [Git Style Guide](#git-style-guide)
  * [Go Style Guide](#go-style-guide)

[Code of Conduct](#code-of-conduct)

## Ask a Question

We will have a public forum soon where you can come and ask questions and have a discussion. For now please open an Issue on GitHub with the label `question`.


## Getting Started

Follow our [Installation Guide](https://github.com/nginxinc/kubernetes-ingress/blob/master/docs/installation.md) to get the NGINX Ingress Controller up and running.

Read the [documentation](https://github.com/nginxinc/kubernetes-ingress/tree/master/docs) and [configuration](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples) examples

### Project Structure

* This Ingress Controller is written in Go and supports both the open source NGINX software and NGINX Plus.
* The project follows a standard Go project layout
    * The main code is found at `cmd/nginx-ic/`
    * The internal code is found at `internal/`
    * Build files for Docker and CI are found under `build/`
    * Deployment yaml files, and Helm files are found at `deployments/`
    * The project dependencies are found at `vendor/`. We use  [dep](https://github.com/golang/dep) for managing dependencies.

## Contributing

### Report a Bug

To report a bug, open an issue on GitHub with the label `bug` using the available bug report issue template. Please ensure the issue has not already been reported.

### Suggest an Enhancement

To suggest an enhancement, please create an issue on GitHub with the label `enhancement` using the available feature issue template.

### Open a Pull Request

* Fork the repo, create a branch, submit a PR when your changes are tested and ready for review
* Fill in [our pull request template](https://github.com/nginxinc/kubernetes-ingress/blob/master/.github/PULL_REQUEST_TEMPLATE.md)

Note: if you’d like to implement a new feature, please consider creating a feature request issue first to start a discussion about the feature.

## Style Guides

### Git Style Guide

* Keep a clean, concise and meaningful git commit history on your branch, rebasing locally and squashing before submitting a PR
* Follow the guidelines of writing a good commit message as described here https://chris.beams.io/posts/git-commit/ and summarised in the next few points
    * In the subject line, use the present tense ("Add feature" not "Added feature")
    * In the subject line, use the imperative mood ("Move cursor to..." not "Moves cursor to...")
    * Limit the subject line to 72 characters or less
    * Reference issues and pull requests liberally after the subject line
    * Add more detailed description in the body of the git message (`git commit -a` to give you more space and time in your text editor to write a good message instead of `git commit -am`)

### Go Style Guide

* Run `gofmt` over your code to automatically resolve a lot of style issues. Most editors support this running automatically when saving a code file.
* Run `go lint` and `go vet` on your code too to catch any other issues.
* Follow this guide on some good practice and idioms for Go -  https://github.com/golang/go/wiki/CodeReviewComments

## Code of Conduct

This project and everyone participating in it is governed by this code.

### Our Pledge

In the interest of fostering an open and welcoming environment, we as
contributors and maintainers pledge to making participation in our project and
our community a harassment-free experience for everyone, regardless of age, body
size, disability, ethnicity, sex characteristics, gender identity and expression,
level of experience, education, socio-economic status, nationality, personal
appearance, race, religion, or sexual identity and orientation.

### Our Standards

Examples of behavior that contributes to creating a positive environment
include:

* Using welcoming and inclusive language
* Being respectful of differing viewpoints and experiences
* Gracefully accepting constructive criticism
* Focusing on what is best for the community
* Showing empathy towards other community members

Examples of unacceptable behavior by participants include:

* The use of sexualized language or imagery and unwelcome sexual attention or
  advances
* Trolling, insulting/derogatory comments, and personal or political attacks
* Public or private harassment
* Publishing others' private information, such as a physical or electronic
  address, without explicit permission
* Other conduct which could reasonably be considered inappropriate in a
  professional setting

### Our Responsibilities

Project maintainers are responsible for clarifying the standards of acceptable
behavior and are expected to take appropriate and fair corrective action in
response to any instances of unacceptable behavior.

Project maintainers have the right and responsibility to remove, edit, or
reject comments, commits, code, wiki edits, issues, and other contributions
that are not aligned to this Code of Conduct, or to ban temporarily or
permanently any contributor for other behaviors that they deem inappropriate,
threatening, offensive, or harmful.

### Scope

This Code of Conduct applies both within project spaces and in public spaces
when an individual is representing the project or its community. Examples of
representing a project or community include using an official project e-mail
address, posting via an official social media account, or acting as an appointed
representative at an online or offline event. Representation of a project may be
further defined and clarified by project maintainers.

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be
reported by contacting the project team at [mailto:nginx@nginx.org]. All
complaints will be reviewed and investigated and will result in a response that
is deemed necessary and appropriate to the circumstances. The project team is
obligated to maintain confidentiality with regard to the reporter of an incident.
Further details of specific enforcement policies may be posted separately.

Project maintainers who do not follow or enforce the Code of Conduct in good
faith may face temporary or permanent repercussions as determined by other
members of the project's leadership.

### Attribution

This Code of Conduct is adapted from the [Contributor Covenant][homepage], version 1.4,
available at https://www.contributor-covenant.org/version/1/4/code-of-conduct.html

[homepage]: https://www.contributor-covenant.org
