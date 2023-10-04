build_dir := ./.build
IMAGE_NAME ?= "cubahno/connexions"
VOLUME_NAME ?= "connexions"
VERSION ?= "latest"

MIN_COVERAGE = 99

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
test:
	go test -race -coverpkg=$(go list ./... | grep -v /examples/) -coverprofile .testCoverage.txt -count=1 ./...

.PHONY: test-integration
test-integration:
	@go test -race -tags=integration -count=1

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

.PHONY: build
build: clean
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${build_dir}/server/bootstrap ./cmd/server/main.go
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${build_dir}/simplifier/bootstrap ./cmd/simplifier/main.go

.PHONY: docker-build
docker-build:
	@docker build --no-cache . \
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
		$(IMAGE_NAME) api

@PHONY: docker-shell
docker-shell:
	@docker run -it --rm \
		$(IMAGE_NAME) bash

@PHONY: tag-next
tag-next:
	@./cmd/tag-next.sh

@PHONY: deploy-docs
deploy-docs:
	mkdocs gh-deploy --force

@PHONY: simplify-schemas
simplify-schemas:
	@go run ./cmd/simplifier/main.go --src=$(src) --dst=$(dst) --replace=$(replace)
