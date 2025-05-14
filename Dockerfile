FROM golang:1.24-alpine AS builder

LABEL maintainer="MS <calebb.jnr`@gmail.com>"

WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .
COPY .env .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/ms/.

# # Use a minimal alpine image for the final stage
# FROM alpine:latest

# # Set the working directory inside the container
# WORKDIR /app/

# # Copy the pre-built binary file from the previous stage
# COPY --from=builder /app/main .

# Expose the port the app runs on
EXPOSE 2345

# Command to run the executable
CMD ["./main"]