
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
