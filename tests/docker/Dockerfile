FROM python:3.9-slim

RUN apt-get update && apt-get install -y curl \
	&& curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
	&& chmod +x ./kubectl \
	&& mv ./kubectl /usr/local/bin

WORKDIR /workspace/tests

COPY tests/requirements.txt /workspace/tests/ 
RUN pip install -r requirements.txt 

COPY tests /workspace/tests
COPY deployments /workspace/deployments

ENTRYPOINT ["python3", "-m", "pytest"]
