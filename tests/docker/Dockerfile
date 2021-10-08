# syntax=docker/dockerfile:1.3
FROM python:3.10

RUN apt-get update \
	&& apt-get install -y curl git \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /workspace/tests

COPY tests/requirements.txt /workspace/tests/
RUN pip install -r requirements.txt

COPY deployments /workspace/deployments

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
	&& install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

COPY tests /workspace/tests

ENTRYPOINT ["python3", "-m", "pytest"]
