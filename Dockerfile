FROM golang:1.13-alpine AS build

RUN apk --no-cache add --update git
RUN mkdir -p /go/src/github.com/arbourd/concourse-slack-alert-resource

ENV CGO_ENABLED 0
ENV GO111MODULE on
WORKDIR /go/src/github.com/arbourd/concourse-slack-alert-resource

COPY go.mod go.sum /go/src/github.com/arbourd/concourse-slack-alert-resource/
RUN go mod download

COPY . /go/src/github.com/arbourd/concourse-slack-alert-resource
RUN go build -o /check github.com/arbourd/concourse-slack-alert-resource/check
RUN go build -o /in github.com/arbourd/concourse-slack-alert-resource/in
RUN go build -o /out github.com/arbourd/concourse-slack-alert-resource/out

FROM alpine:3.10
RUN apk add --no-cache ca-certificates

COPY --from=build /check /opt/resource/check
COPY --from=build /in /opt/resource/in
COPY --from=build /out /opt/resource/out
