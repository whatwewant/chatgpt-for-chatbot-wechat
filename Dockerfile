# Builder
FROM whatwewant/builder-go:v1.20-1 as builder

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

RUN GOOS=linux \
  GOARCH=amd64 \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o chatgpt-for-chatbot-wechat

# Server
FROM whatwewant/go:v1.20-1

LABEL MAINTAINER="Zero<tobewhatwewant@gmail.com>"

LABEL org.opencontainers.image.source="https://github.com/go-zoox/chatgpt-for-chatbot-wechat"

ARG VERSION=latest

ENV MODE=production

COPY --from=builder /build/chatgpt-for-chatbot-wechat /bin

ENV VERSION=${VERSION}

CMD chatgpt-for-chatbot-wechat
