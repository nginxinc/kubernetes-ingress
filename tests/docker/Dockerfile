# syntax=docker/dockerfile:1.4
# this is here so we can grab the latest version of kind and have dependabot keep it up to date
FROM kindest/node:v1.23.5

FROM python:3.10

RUN apt-get update \
	&& apt-get install -y curl git \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /workspace/tests

COPY --link tests/requirements.txt /workspace/tests/
RUN pip install -r requirements.txt

COPY --link deployments /workspace/deployments

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
	&& install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl \
	&& apt-get update && apt-get install -y apache2-utils

COPY --link tests /workspace/tests

ENTRYPOINT ["python3", "-m", "pytest"]
