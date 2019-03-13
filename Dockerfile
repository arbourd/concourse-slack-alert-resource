FROM golang:1.12-alpine AS build

ENV CGO_ENABLED 0
RUN mkdir -p /go/src/github.com/arbourd/concourse-slack-alert-resource
WORKDIR /go/src/github.com/arbourd/concourse-slack-alert-resource
COPY . /go/src/github.com/arbourd/concourse-slack-alert-resource

RUN go build -o /check github.com/arbourd/concourse-slack-alert-resource/check
RUN go build -o /in github.com/arbourd/concourse-slack-alert-resource/in
RUN go build -o /out github.com/arbourd/concourse-slack-alert-resource/out

FROM alpine:3.9
RUN apk add --no-cache ca-certificates

COPY --from=build /check /opt/resource/check
COPY --from=build /in /opt/resource/in
COPY --from=build /out /opt/resource/out
