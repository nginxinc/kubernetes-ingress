FROM python:3.9-slim

ARG GCLOUD_VERSION=338.0.0

RUN apt-get update && apt-get install -y curl \
	&& curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
	&& chmod +x ./kubectl \
	&& mv ./kubectl /usr/local/bin \
	&& curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_VERSION}-linux-x86_64.tar.gz \
    && tar xvzf google-cloud-sdk-${GCLOUD_VERSION}-linux-x86_64.tar.gz \
    && mv google-cloud-sdk /usr/lib/

WORKDIR /workspace/tests

COPY tests/requirements.txt /workspace/tests/ 
RUN pip install -r requirements.txt 

COPY tests /workspace/tests
COPY deployments /workspace/deployments

ENV PATH="/usr/lib/google-cloud-sdk/bin:${PATH}"

ENTRYPOINT ["python3", "-m", "pytest"]
