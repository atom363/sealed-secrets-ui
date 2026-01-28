FROM golang:1.25.6-bookworm AS build
WORKDIR /go/src/app
COPY . ./

ARG VERSION

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go mod tidy \
  && go build -o /go/bin/app -ldflags="-s -w"

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /ca-certificates.crt
COPY --from=build /go/bin/app /

CMD ["/app"]
