FROM golang:1.25.5 AS builder
ENV CGO_ENABLED=1
RUN apt-get install -y git make

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor/ ./vendor/
COPY . .

# Build tools (will be included in final image for user projects)
RUN go build -mod=vendor -o /app/.build/gen-discover ./cmd/gen/discover
RUN go build -mod=vendor -o /app/.build/gen-service ./cmd/gen/service

# Generate service imports for connexions itself
RUN /app/.build/gen-discover

# Build server using vendor (no need to download modules)
RUN go build -mod=vendor -o /app/.build/server/bootstrap ./cmd/server

# Get version
RUN git describe --tags --abbrev=0 > version.txt || echo "dev" > version.txt

FROM golang:1.25.5
ENV CGO_ENABLED=1

WORKDIR /app

COPY --from=builder /app/.build/server/bootstrap /usr/local/bin/api
COPY --from=builder /app/.build/gen-discover /usr/local/bin/gen-discover
COPY --from=builder /app/.build/gen-service /usr/local/bin/gen-service
COPY --from=builder /app/go.mod /app/go.mod
COPY --from=builder /app/go.sum /app/go.sum
COPY --from=builder /app/vendor /app/vendor
COPY --from=builder /app/cmd /app/cmd
COPY --from=builder /app/pkg /app/pkg
COPY --from=builder /app/internal /app/internal
COPY --from=builder /app/resources /app/resources
COPY --from=builder /app/version.txt /app/version.txt

RUN export APP_VERSION=$(cat /app/version.txt) && \
    echo "APP_VERSION=$APP_VERSION" >> /app/.env

# Create directories for user data
# Users mount their files here and services are auto-generated on startup:
#   -v ~/my-specs:/app/resources/data/openapi    (OpenAPI specs → auto-generate services)
#   -v ~/my-static:/app/resources/data/static    (static files → auto-generate services)
# Generated services are stored in /app/resources/data/services
RUN mkdir -p /app/resources/data/services
RUN mkdir -p /app/resources/data/openapi
RUN mkdir -p /app/resources/data/static

COPY resources/ui /app/resources/ui
COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /usr/local/bin/api
RUN chmod +x /usr/local/bin/gen-discover
RUN chmod +x /usr/local/bin/gen-service
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["api"]
