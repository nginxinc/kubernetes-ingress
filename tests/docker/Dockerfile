FROM python:3.9.4-slim

RUN mkdir /workspace

WORKDIR /workspace

COPY tests tests
COPY deployments deployments

WORKDIR /workspace/tests
RUN apt-get update && apt-get install -y curl && \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl ; mv ./kubectl /usr/local/bin ; pip install -r requirements.txt ; rm -rf ‘.pytest_cache’

ENTRYPOINT ["python3", "-m", "pytest"]