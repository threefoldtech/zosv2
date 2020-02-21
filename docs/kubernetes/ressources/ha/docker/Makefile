# Use some sensible default shell settings
SHELL := /bin/bash -o pipefail
.SILENT:
.DEFAULT_GOAL := help

# Kubernetes configuration
KUBE_MASTER_VIP   ?= 192.168.168.100
KUBE_MASTER_IP_01 ?= 192.168.168.101
KUBE_MASTER_IP_02 ?= 192.168.168.102
KUBE_MASTER_IP_03 ?= 192.168.168.103

KUBE_MASTER_NAME_01 ?= master1
KUBE_MASTER_NAME_02 ?= master2
KUBE_MASTER_NAME_03 ?= master3

##@ launch
.PHONY: launch
launch: down base-build up-m1 start-ha-m1 up-m2 up-m3 start-ha up-agent print kub   ## launch the HA cluster creation


##@ start-ha-m1
.PHONY: start-ha-m1  
start-ha-m1: ## start the HAProxy and keepalived services on master node 1
	docker exec $(KUBE_MASTER_NAME_01) /home/scripts/service_launch.sh 
	sleep 5 

##@ print
.PHONY: print
print: ## print kubernetes IP
	sed -i "s/127.0.0.1:6443/$(KUBE_MASTER_VIP):8443/" kubeconfig.yaml
	echo "Created a Kubernetes:"
	echo "- Control Plane Endpoint: $(KUBE_MASTER_VIP)"
	echo "- Master IP 01: $(KUBE_MASTER_IP_01)"
	echo "- Master IP 02: $(KUBE_MASTER_IP_02)"
	echo "- Master IP 03: $(KUBE_MASTER_IP_03)"
	echo "......"
	sleep 12

##@ kub
.PHONY: kub
kub: ## kubectl get nodes
	kubectl --kubeconfig=kubeconfig.yaml get nodes

##@ start-ha
.PHONY: start-ha  
start-ha: ## start the HAProxy and keepalived services on master nodes
	sleep 2
	docker exec $(KUBE_MASTER_NAME_02) /home/scripts/service_launch.sh
	sleep 2
	docker exec $(KUBE_MASTER_NAME_03) /home/scripts/service_launch.sh

##@ base-build
.PHONY: base-build  
base-build: ## Build the docker base image
	docker build -t updated-bionic -f dockerfile-base-update-bionic . 

##@ up-m1  
.PHONY: up-m1 
up-m1:  ## Launch the docker-compose up command for master1
	docker-compose up -d $(KUBE_MASTER_NAME_01)
	sleep 5

##@ up-m2
.PHONY: up-m2 
up-m2:  ## Launch the docker-compose up command for master2
	sleep 5
	docker-compose up -d $(KUBE_MASTER_NAME_02)


##@ up-m3 
.PHONY: up-m3 
up-m3:  ## Launch the docker-compose up command for master3
	sleep 2
	docker-compose up -d $(KUBE_MASTER_NAME_03)


##@ up-agent  
.PHONY: up-agent  
up-agent:  ## Launch the docker-compose up command for agent
	docker-compose up -d agent

##@ up 
.PHONY: up  
up:  ## Launch the docker-compose up command
	docker-compose up -d

##@ down 
.PHONY: down 
down:  ## Launch the docker-compose down command
	docker-compose down

##@ Misc
.PHONY: help
help: ## Display this help
	awk \
	  'BEGIN { \
	    FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n" \
	  } /^[a-zA-Z_-]+:.*?##/ { \
	    printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 \
	  } /^##@/ { \
	    printf "\n\033[1m%s\033[0m\n", substr($$0, 5) \
	  }' $(MAKEFILE_LIST)

