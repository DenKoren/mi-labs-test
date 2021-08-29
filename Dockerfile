# Build container
FROM golang:1.16-alpine AS build-env
RUN apk add build-base git

ENV GOROOT /usr/local/go
ENV GOPATH /go
ENV PATH $PATH:$GOPATH/bin

# Download dependencies list to speed-up rebuild process
COPY ./go.mod /mi-labs-test/go.mod
WORKDIR /mi-labs-test
RUN go mod download

# Compile Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Build project
COPY . /mi-labs-test
WORKDIR /mi-labs-test/cmd/zapuskator
RUN go build -gcflags="all=-N -l" -o /server

# Run container
FROM alpine

EXPOSE 4334 4224 2345

COPY --from=build-env /server /server
COPY --from=build-env /go/bin/dlv /dlv

WORKDIR /
CMD ["/dlv", "--listen=:2345", "--headless=true", "--api-version=2", "exec", "/server"]
