FROM golang as builder

COPY . /app
WORKDIR /app

RUN go install -v .

FROM debian:stable-slim
RUN apt update \
  && apt install -y ca-certificates \
  && rm -fr /var/lib/apt/lists/*
COPY --from=builder /go/bin/jwt-auth-subreq /app
ENTRYPOINT [ "/app" ]