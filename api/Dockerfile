FROM golang:1.18-alpine as builder
RUN apk --no-cache add gcc g++

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -a ./...
RUN go test -race ./...

FROM builder as installer
RUN GOBIN=/server go install -a ./...

FROM golang:1.18-alpine as runner

COPY --from=installer /server /server

RUN addgroup -S -g 1001 non-root-group
RUN adduser -S -u 1001 -G non-root-group non-root-user
USER 1001

ENTRYPOINT ["/server/catalogue"]
