FROM golang:1.23 as builder

RUN apt-get update && apt-get install -y git nano

WORKDIR /app
COPY . .
RUN make build
RUN git describe --tags --abbrev=0 > version.txt

FROM golang:1.23
WORKDIR /app
COPY --from=builder /app/.build/server/bootstrap /usr/local/bin/api
COPY --from=builder /app/.build/simplifier/bootstrap /usr/local/bin/simplify-schemas
COPY --from=builder /app/version.txt /app/resources/version.txt

RUN export APP_VERSION=$(cat /app/resources/version.txt) && \
    echo "APP_VERSION=$APP_VERSION" >> /app/.env

COPY resources/ui /app/resources/ui
COPY resources/samples /app/resources/data/services
COPY resources/contexts /app/resources/data/contexts
COPY resources/openapi.yml /app/resources/openapi.yml

COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /usr/local/bin/api
RUN chmod +x entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["api"]
