FROM golang:1.24.2 AS builder
ENV CGO_ENABLED=1
RUN apt-get install -y git make

WORKDIR /app
COPY . .
RUN make build
RUN git describe --tags --abbrev=0 > version.txt

FROM golang:1.24.2
ENV CGO_ENABLED=1

WORKDIR /app

COPY --from=builder /app/.build/server/bootstrap /usr/local/bin/api
COPY --from=builder /app/.build/simplifier/bootstrap /usr/local/bin/simplify-schemas
COPY --from=builder /app/version.txt /app/resources/version.txt

RUN export APP_VERSION=$(cat /app/resources/version.txt) && \
    echo "APP_VERSION=$APP_VERSION" >> /app/.env

COPY resources/contexts /app/resources/data/contexts
COPY resources/openapi.yml /app/resources/openapi.yml
COPY resources/samples /app/resources/data/services
COPY resources/ui /app/resources/ui

COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /usr/local/bin/api
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["api"]
