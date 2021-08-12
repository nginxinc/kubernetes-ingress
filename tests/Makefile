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
KIND_KUBE_CONFIG_FOLDER = $${HOME}/.kube/kind
SHOW_IC_LOGS = no
PYTEST_ARGS =
DOCKERFILEPATH = docker/Dockerfile

.PHONY: build
build:
	docker build -t $(PREFIX):$(TAG) -f $(DOCKERFILEPATH) ..

.PHONY: run-tests
run-tests:
	docker run --rm -v $(KUBE_CONFIG_FOLDER):/root/.kube $(PREFIX):$(TAG) --context=$(CONTEXT) --image=$(BUILD_IMAGE) --image-pull-policy=$(PULL_POLICY) --deployment-type=$(DEPLOYMENT_TYPE) --ic-type=$(IC_TYPE) --service=$(SERVICE) --node-ip=$(NODE_IP) --show-ic-logs=$(SHOW_IC_LOGS) $(PYTEST_ARGS)

.PHONY: run-tests-in-kind
run-tests-in-kind:
	$(eval KIND_CLUSTER_IP=$(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane))
	sed -i "" "s|server:.*|server: https://$(KIND_CLUSTER_IP):6443|" $(KIND_KUBE_CONFIG_FOLDER)/config
	docker run --network=kind --rm -v $(KIND_KUBE_CONFIG_FOLDER):/root/.kube $(PREFIX):$(TAG) --context=kind-kind --image=$(BUILD_IMAGE) --image-pull-policy=$(PULL_POLICY) --deployment-type=$(DEPLOYMENT_TYPE) --ic-type=$(IC_TYPE) --service=nodeport --node-ip=$(KIND_CLUSTER_IP) --show-ic-logs=$(SHOW_IC_LOGS) $(PYTEST_ARGS)

.PHONY: create-kind-cluster
create-kind-cluster:
	kind create cluster
	kind export kubeconfig --kubeconfig $(KIND_KUBE_CONFIG_FOLDER)/config

.PHONY: delete-kind-cluster
delete-kind-cluster:
	kind delete cluster