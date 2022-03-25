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
KIND_TEMPLATE_FILE_PATH=kind/kind-config-template.yaml
IP_FAMILY ?= ipv4
IP_ADDR_PROPERTY = .IPAddress
IP_ADDR_FMT = "%s"
API_SERVER_ADDR = 127.0.0.1


ifeq (${IP_FAMILY}, ipv6)
IP_ADDR_PROPERTY = .GlobalIPv6Address
IP_ADDR_FMT = "[%s]"
endif

define PRINT_HELP_PYSCRIPT
import re, sys

for line in sys.stdin:
	match = re.match(r'^([a-zA-Z_-]+):.*?## (.*)$$', line)
	if match:
		target, help = match.groups()
		print("%-20s %s" % (target, help))
endef
export PRINT_HELP_PYSCRIPT

default: help

.PHONY: help ## Show this help
help: ## Show available make targets
	@python -c "$$PRINT_HELP_PYSCRIPT" < $(MAKEFILE_LIST)


.PHONY: build
build: ## Run build
	docker build -t $(PREFIX):$(TAG) -f $(DOCKERFILEPATH) ..


.PHONY: run-tests
run-tests: ## Run tests
	docker run --rm -v $(KUBE_CONFIG_FOLDER):/root/.kube $(PREFIX):$(TAG) --context=$(CONTEXT) --image=$(BUILD_IMAGE) --image-pull-policy=$(PULL_POLICY) --deployment-type=$(DEPLOYMENT_TYPE) --ic-type=$(IC_TYPE) --service=$(SERVICE) --node-ip=$(NODE_IP) --show-ic-logs=$(SHOW_IC_LOGS) $(PYTEST_ARGS)


.PHONY: get-kind-cluster-ip ## Get Kind cluster IP Address
get-kind-cluster-ip:	
	$(eval KIND_CLUSTER_IP=$(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{${IP_ADDR_PROPERTY}}}{{end}}' kind-control-plane ))
	$(eval KIND_CLUSTER_IP=$(shell printf ${IP_ADDR_FMT} "${KIND_CLUSTER_IP}"))


.PHONY: update-test-kind-config ## Update Kind config
update-test-kind-config: get-kind-cluster-ip
	sed -ir "s|server:.*|server: https://${KIND_CLUSTER_IP}:6443|" $(KIND_KUBE_CONFIG_FOLDER)/config


.PHONY: run-tests-in-kind
run-tests-in-kind: update-test-kind-config ## Run tests in Kind
	docker run --network=kind --rm -v $(KIND_KUBE_CONFIG_FOLDER):/root/.kube $(PREFIX):$(TAG) \
 		--context=kind-kind \
 		--image=$(BUILD_IMAGE) --image-pull-policy=$(PULL_POLICY) \
 		--deployment-type=$(DEPLOYMENT_TYPE) \
 		--ic-type=$(IC_TYPE) \
 		--service=nodeport \
 		--node-ip=$(KIND_CLUSTER_IP) \
 		--show-ic-logs=$(SHOW_IC_LOGS) \
 		$(PYTEST_ARGS)


.PHONY: create-kind-cluster
create-kind-cluster: ## Create Kind cluster
	$(eval KIND_IMAGE=$(shell grep -m1 'FROM kindest/node' <docker/Dockerfile | awk -F'[ ]' '{print $$2}'))
	cat ${KIND_TEMPLATE_FILE_PATH} | sed "s/{{.IPFamily}}/${IP_FAMILY}/" | sed "s/{{.APIServerAddress}}/${API_SERVER_ADDR}/"  > kind-config.yaml
	kind create cluster --image $(KIND_IMAGE) --config kind-config.yaml
	kind export kubeconfig --kubeconfig $(KIND_KUBE_CONFIG_FOLDER)/config


.PHONY: delete-kind-cluster
delete-kind-cluster: ## Delete Kind cluster
	kind delete cluster
