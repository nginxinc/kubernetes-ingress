# syntax=docker/dockerfile:1.5
# this is here so we can grab the latest version of kind and have dependabot keep it up to date
FROM kindest/node:v1.29.1@sha256:a0cc28af37cf39b019e2b448c54d1a3f789de32536cb5a5db61a49623e527144

FROM python:3.12@sha256:3733015cdd1bd7d9a0b9fe21a925b608de82131aa4f3d397e465a1fcb545d36f

RUN apt-get update \
	&& apt-get install -y curl git \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /workspace/tests

COPY --link tests/requirements.txt /workspace/tests/
RUN pip install --require-hashes -r requirements.txt --no-deps

COPY --link deployments /workspace/deployments
COPY --link config /workspace/config

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
	&& install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl \
	&& apt-get update && apt-get install -y apache2-utils

COPY --link tests /workspace/tests

COPY --link pyproject.toml /workspace/

ENTRYPOINT ["python3", "-m", "pytest"]
