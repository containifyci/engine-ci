FROM hashicorp/packer:full as builder

FROM golang:1.27rc2-alpine

COPY --from=builder /bin/packer /bin/packer

RUN packer plugins install github.com/hetznercloud/hcloud