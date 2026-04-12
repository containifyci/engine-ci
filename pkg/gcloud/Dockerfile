FROM golang:1.26.2-alpine as builder

WORKDIR /src

ADD src/ .

RUN go generate main.go && \
  go build -o oidc main.go

FROM alpine

WORKDIR /

COPY --from=builder /src/oidc /bin/oidc

CMD /bin/oidc
