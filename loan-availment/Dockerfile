# # Start with the official Golang Alpine image
# FROM golang:1.22.4-alpine

# # Install dependencies required for building with CGO and librdkafka
# RUN apk add --no-cache gcc musl-dev libc-dev librdkafka-dev

# # Set the Current Working Directory inside the container
# WORKDIR /app

# # Enable CGO
# ENV CGO_ENABLED=1

# # Set the necessary build tags (for Alpine musl, use `musl`)
# ENV GOFLAGS="-tags=musl"

# # Copy go mod and sum files
# COPY go.mod go.sum ./

# # Copy the source code into the container
# COPY . .

# # Build the Go application
# RUN go build -o main cmd/main.go

# # Expose port 80 to the outside world
# EXPOSE 8080

# # Command to run the Go application
# CMD ["./main"]

FROM golang:1.24.2-alpine as build

# Set the Current Working Directory inside the container
WORKDIR /app

# Install dependencies required for building with CGO and librdkafka
RUN apk add --no-cache gcc musl-dev libc-dev librdkafka-dev

# Copy go.mod and go.sum files (if needed, can be omitted if not using go mod)
COPY go.mod go.sum ./

# Download dependencies without go mod
RUN go get -d -v ./...

# Install the package
RUN go install -v ./...

# Copy the source code into the container
COPY . .

# Enable CGO
ENV CGO_ENABLED=1

# Force confluent-kafka-go to link against system-installed librdkafka instead of vendored static archive
ENV GOFLAGS="-tags=dynamic"

# Build the Go app (binary will be named after the containing directory, e.g., `app`)
RUN go build cmd/main.go

# Stage 2: Create a minimal Docker image to run the Go application
FROM alpine:3.21
RUN apk add --no-cache librdkafka && apk upgrade --no-cache
# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file and .env from the build stage
COPY --from=build /app/main .  
# COPY --from=build /app/.env .

ENTRYPOINT ["/app/main"]
