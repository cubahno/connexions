FROM golang:1.21 as builder

WORKDIR /app
COPY . .
RUN make build

FROM golang:1.21
WORKDIR /app
COPY --from=builder /app/.build/server/bootstrap /usr/local/bin/api

COPY resources/ui /app/resources/ui
COPY resources/samples /app/resources/data/services
COPY resources/contexts /app/resources/data/contexts
COPY resources/openapi.yml /app/resources/openapi.yml

COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /usr/local/bin/api
RUN chmod +x entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["api"]
