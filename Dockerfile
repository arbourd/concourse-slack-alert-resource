FROM golang:1.24-alpine AS build
WORKDIR /go/src/github.com/arbourd/concourse-slack-alert-resource
RUN apk --no-cache add --update git

COPY go.* ./
RUN go mod download

COPY . ./
RUN go build -o /check github.com/arbourd/concourse-slack-alert-resource/check
RUN go build -o /in github.com/arbourd/concourse-slack-alert-resource/in
RUN go build -o /out github.com/arbourd/concourse-slack-alert-resource/out

FROM alpine:3.19
RUN apk add --no-cache ca-certificates

COPY --from=build /check /opt/resource/check
COPY --from=build /in /opt/resource/in
COPY --from=build /out /opt/resource/out
