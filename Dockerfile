# Build the matterwick
ARG DOCKER_BUILD_IMAGE=golang:1.15.2
ARG DOCKER_BASE_IMAGE=alpine:3.12

FROM ${DOCKER_BUILD_IMAGE} AS build
WORKDIR /oncall/
COPY . /oncall/
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build .

# Final Image
FROM ${DOCKER_BASE_IMAGE}


ENV ONCALL=/app/opsgenie \
  USER_UID=10001 \
  USER_NAME=oncall

WORKDIR /app/

RUN  apk update && apk add ca-certificates

COPY --from=build /oncall/opsgenie /app/
COPY --from=build /oncall/assets/* /app/assets/
COPY --from=build /oncall/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
