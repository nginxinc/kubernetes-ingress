FROM python:3.7.2-slim

RUN mkdir /workspace

WORKDIR /workspace

COPY tests tests
COPY deployments deployments

WORKDIR /workspace/tests
RUN apt-get install -y curl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin
RUN pip install -r requirements.txt
RUN rm -rf '.pytest_cache'

ENTRYPOINT ["python3", "-m", "pytest"]