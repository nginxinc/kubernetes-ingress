# syntax=docker/dockerfile:1.4
FROM python:3.11

ARG HELM_VERSION=3.5.4

RUN apt-get update && apt-get install -y curl git jq apache2-utils apt-transport-https ca-certificates gnupg \
	&& curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
	&& chmod +x ./kubectl \
	&& mv ./kubectl /usr/local/bin \
	&& echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
	&& curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | tee /usr/share/keyrings/cloud.google.gpg \
	&& apt-get update && apt-get install google-cloud-cli \
	&& apt-get install google-cloud-sdk-gke-gcloud-auth-plugin \
	&& curl -LO https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz \
	&& tar -zxvf helm-v${HELM_VERSION}-linux-amd64.tar.gz \
	&& mv linux-amd64/helm /usr/local/bin/helm

WORKDIR /workspace/tests

COPY --link tests/requirements.txt /workspace/tests/
RUN python -m ensurepip --upgrade
RUN pip install --require-hashes -r requirements.txt

COPY --link tests /workspace/tests
COPY --link deployments /workspace/deployments

ENTRYPOINT ["python3", "-m", "pytest"]
