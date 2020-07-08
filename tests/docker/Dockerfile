FROM python:3.7.2-slim

RUN mkdir /workspace

WORKDIR /workspace

COPY tests tests
COPY deployments deployments

WORKDIR /workspace/tests

RUN pip install -r requirements.txt
RUN rm -rf '.pytest_cache'

ENTRYPOINT [ "pytest", "-m", "\"not ap\""]