FROM golang:1.21 as builder

WORKDIR /app
COPY . .
RUN make build

FROM golang:1.21
WORKDIR /app
COPY --from=builder /app/.build/server/bootstrap /usr/local/bin/api
COPY resources /app/resources
COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /usr/local/bin/api
RUN chmod +x entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["api"]
