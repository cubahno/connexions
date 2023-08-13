build_dir := ./.build
package_dir := ${build_dir}
IMAGE_NAME ?= "cubahno/connexions"
VOLUME_NAME ?= "connexions"

define docker-cmd
	sh -c 'docker-compose --env-file=.env.dist run --rm -e app_env=testing app $(1)'
endef

.PHONY: test
test:
	@if [ "$(with_docker)" = "true" ]; then \
		$(call docker-cmd, sh -c "/wait && go test -race ./... -coverprofile .testCoverage.txt -count=1"); \
	else \
		go test -race ./... -coverprofile .testCoverage.txt -count=1; \
	fi;

.PHONY: clean
clean:
	rm -rf ${build_dir}

.PHONY: build
build: clean
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${build_dir}/server/bootstrap ./.cli/server/main.go

	mkdir -p ${build_dir}/server/resources

	cp -r ./resources/* ${build_dir}/server/resources/
	cp ./resources/config.yml.dist ${build_dir}/server/resources/config.yml

.PHONY: docker-build
docker-build:
	@docker build . \
		--tag $(IMAGE_NAME):latest \
		--tag $(IMAGE_NAME):$(VERSION)

.PHONY: docker-push
docker-push:
	@docker push $(IMAGE_NAME):$(VERSION)
	@docker push $(IMAGE_NAME):latest

.PHONY: docker-run
docker-run:
	@docker run -it --rm \
		-p 2200:2200 \
		-v $(VOLUME_NAME):/app/resources \
		--name $(IMAGE_NAME) \
		$(IMAGE_NAME) api

@PHONY: docker-shell
docker-shell:
	@docker run -it --rm \
		-v connexions:/app/resources \
		$(IMAGE_NAME) bash

@PHONY: tag-next
tag-next:
	@git fetch --tags
	@TAG=$$(git describe --abbrev=0 --tags)
	@NEXT_TAG=$$(echo $${TAG%.*}.$$(($${TAG##*.} + 1)))
	@echo "Tagging and pushing $$NEXT_TAG"
	@git tag $$NEXT_TAG
	@git push origin $$NEXT_TAG
