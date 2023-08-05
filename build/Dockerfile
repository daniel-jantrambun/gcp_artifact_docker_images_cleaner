FROM golang:1.20

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /gcp_docker_images_cleaner

# Run
CMD ["/gcp_artifact_docker_images_cleaner"]