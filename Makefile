PROJECT_ID = audiube
IMAGE_VERSION = v1.4
SERVICE_NAME = audiube
IMAGE_NAME := $(SERVICE_NAME):$(IMAGE_VERSION)
GCR_IMAGE_TAG := gcr.io/$(PROJECT_ID)/$(IMAGE_NAME)

build: Dockerfile
	go build -o /tmp/dummy . && \
	rm /tmp/dummy && \
	docker build -t $(IMAGE_NAME) .

list-images:
	docker images | grep $(SERVICE_NAME)

push:
	docker tag $(IMAGE_NAME) $(GCR_IMAGE_TAG) && \
	gcloud docker -- push $(GCR_IMAGE_TAG)

init-deploy: kubernetes/deployment.yml kubernetes/service.yml
	kubectl apply -f kubernetes/deployment.yml
	kubectl apply -f kubernetes/service.yml

apply: kubernetes/deployment.yml kubernetes/service.yml
	kubectl apply -f kubernetes/deployment.yml
	kubectl apply -f kubernetes/service.yml

apply-cluster:
	kubectl apply -f kubernetes/deployment.yml

apply-service:
	kubectl apply -f kubernetes/service.yml

deploy:
	make build
	make push
	make apply-cluster
