FROM golang:1.15 AS builder
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
ENV GOBIN /bin
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN go build -a -installsuffix nocgo -o /tmp/semver .


FROM alpine
RUN adduser -D -u 1000 semver
COPY --from=builder /tmp/semver /usr/local/bin
USER 1000
ENTRYPOINT [ "semver" ]