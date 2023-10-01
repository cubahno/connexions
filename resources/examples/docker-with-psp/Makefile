IMAGE_NAME ?= "cubahno/connexions-psp"
VERSION ?= "latest"

docker-build:
	@docker build --no-cache . \
		--tag $(IMAGE_NAME):latest \
		--tag $(IMAGE_NAME):$(VERSION)

.PHONY: docker-push
docker-push:
	@docker push $(IMAGE_NAME):$(VERSION)
	@docker push $(IMAGE_NAME):latest
