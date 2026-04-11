FROM node:22-alpine

RUN apk --no-cache add \
      bash git openssh-client curl ca-certificates \
      go \
  && npm install -g @anthropic-ai/claude-code@latest \
  && rm -rf /var/cache/apk/* /root/.npm

WORKDIR /src

RUN go version