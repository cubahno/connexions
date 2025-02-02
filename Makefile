build_dir := ./.build
IMAGE_NAME ?= "cubahno/connexions"
VOLUME_NAME ?= "connexions"
VERSION ?= "latest"

MIN_COVERAGE = 90

define docker-cmd
	sh -c 'docker-compose --env-file=.env.dist run --rm -e app_env=testing app $(1)'
endef

.PHONY: lint
lint:
	go fmt ./...
	golangci-lint run
	go vet ./...
	go mod download && go mod tidy && go mod verify

.PHONY: test
# TODO: add race flag, currently it breaks plugins tests
test:
	@if [ -z "$(PKG)" ]; then \
		go test \
			-coverpkg=$(go list ./internal/...) \
			-coverprofile .testCoverage.txt \
			-count=1 \
			./internal/...; \
	else \
  		go test -coverpkg=$(go list ./... | grep -v /examples/ | grep -v /cmd/) -coverprofile=.testCoverage.txt -count=1 ./$(PKG)/...; \
	fi

.PHONY: fetch-specs
fetch-specs:
	rm -rf ./testdata/specs
	git clone https://github.com/cubahno/specs.git ./testdata/specs
	find ./testdata/specs -mindepth 1 -name ".*" -exec rm -rf {} +

.PHONY: test-integration
test-integration: fetch-specs
	@go test -v -tags=integration ./...

.PHONY: test-with-check-coverage
test-with-check-coverage: test
	@coverage=$$(go tool cover -func=.testCoverage.txt | awk '/^total:/{print $$3}' | tr -d '%'); \
	echo "Code coverage $$coverage%."; \
	if [ "$$(echo "$$coverage < $(MIN_COVERAGE)" | bc -l)" -eq 1 ]; then \
	  echo "Code coverage $$coverage% is less than $(MIN_COVERAGE)%."; \
	  exit 1; \
	fi

.PHONY: clean
clean:
	rm -rf ${build_dir}

.PHONY: clean-cache
clean-cache:
	go clean -cache -modcache -i -r

.PHONY: build
build: clean
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${build_dir}/server/bootstrap ./cmd/server/main.go
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${build_dir}/simplifier/bootstrap ./cmd/simplifier/main.go

.PHONY: docker-build
docker-build:
	@docker build \
		--no-cache . \
		--tag $(IMAGE_NAME):latest \
		--tag $(IMAGE_NAME):$(VERSION)

.PHONY: docker-build-multiarch
docker-build-multiarch:
	@docker buildx build \
		--platform linux/arm64,linux/amd64 \
		--no-cache . \
		--tag $(IMAGE_NAME):latest \
		--tag $(IMAGE_NAME):$(VERSION) \
		--push

.PHONY: docker-push
docker-push:
	@docker push $(IMAGE_NAME):$(VERSION)
	@docker push $(IMAGE_NAME):latest

.PHONY: docker-run
docker-run:
	@docker run -it --rm \
		--platform=linux/arm64 \
		-p 2200:2200 \
		$(IMAGE_NAME) api

@PHONY: docker-shell
docker-shell:
	@docker run -it --rm \
		$(IMAGE_NAME) bash

@PHONY: tag-next
tag-next:
	@./cmd/tag-next.sh

@PHONY: simplify-schemas
simplify-schemas:
	@go run ./cmd/simplifier/main.go --src=$(src) --dst=$(dst) --replace=$(replace)

@PHONY: docs-install
docs-install:
	@pip3 install mkdocs mkdocs-material

@PHONY: docs-prepare
docs-prepare:
	@go run ./cmd/fake_list/main.go

@PHONY: docs-serve
docs-serve: docs-prepare
	mkdocs serve

@PHONY: docs-deploy
docs-deploy: docs-prepare
	mkdocs gh-deploy --force
