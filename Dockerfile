FROM golang as builder

COPY . /app
WORKDIR /app

RUN go install -v .

FROM debian:stable-slim
COPY --from=builder /go/bin/jwt-auth-subreq /app
ENTRYPOINT [ "/app" ]