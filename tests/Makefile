CONTEXT =
BUILD_IMAGE = nginx/nginx-ingress:edge
PULL_POLICY = IfNotPresent
DEPLOYMENT_TYPE = deployment
IC_TYPE = nginx-ingress
SERVICE = nodeport
NODE_IP =
TAG = latest
PREFIX = test-runner
KUBE_CONFIG_FOLDER = $${HOME}/.kube
SHOW_IC_LOGS = no
PYTEST_ARGS =

build:
	docker build -t $(PREFIX):$(TAG) -f docker/Dockerfile ..

run-tests:
	docker run --rm -v $(KUBE_CONFIG_FOLDER):/root/.kube $(PREFIX):$(TAG) --context=$(CONTEXT) --image=$(BUILD_IMAGE) --image-pull-policy=$(PULL_POLICY) --deployment-type=$(DEPLOYMENT_TYPE) --ic-type=$(IC_TYPE) --service=$(SERVICE) --node-ip=$(NODE_IP) --show-ic-logs=$(SHOW_IC_LOGS) $(PYTEST_ARGS)