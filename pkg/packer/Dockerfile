FROM hashicorp/packer:full as builder

FROM golang:1.26.3-alpine

COPY --from=builder /bin/packer /bin/packer

RUN packer plugins install github.com/hetznercloud/hcloud