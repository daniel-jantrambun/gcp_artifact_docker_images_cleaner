FROM golang:1.20.7-alpine3.18 as golang-base

FROM golang-base as builder-base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

FROM builder-base as built
COPY *.go ./
# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /gcp_artifact_docker_images_cleaner

FROM alpine:latest as production
COPY --from=built /gcp_artifact_docker_images_cleaner /gcp_artifact_docker_images_cleaner
# Run
CMD ["/gcp_artifact_docker_images_cleaner"]