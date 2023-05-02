FROM alpine:3.17.3 AS builder
RUN apk add ca-certificates

FROM scratch
COPY ./out/linux-amd64/openstack-check-exporter /
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER nobody
ENTRYPOINT ["/openstack-check-exporter"]