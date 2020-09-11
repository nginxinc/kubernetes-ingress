FROM python:3.7.2-slim

RUN mkdir /workspace

WORKDIR /workspace

COPY tests tests
COPY deployments deployments

WORKDIR /workspace/tests
RUN sudo apt-get update && sudo apt-get install -y kubectl
RUN pip install -r requirements.txt
RUN rm -rf '.pytest_cache'

ENTRYPOINT ["python3", "-m", "pytest"]