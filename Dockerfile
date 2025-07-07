FROM golang:1.24-alpine AS builder

LABEL maintainer="MS <calebb.jnr@gmail.com>"

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc git make musl-dev

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .
COPY .env .

# Build plugins first(example for one plugin, repeat for others as needed)
# RUN mkdir -p /app/plugins
# RUN for f in ./plugins/*/*.go; do \
#       name=$(basename $(dirname "$f")); \
#       go build -buildmode=plugin -o /app/plugins/"$name".so "$f"; \
#     done

# Build all plugins (assuming plugins are in pkg/plugins directory)
RUN mkdir -p /app/plugins && \
    for plugin_dir in pkg/plugins/*; do \
        if [ -d "$plugin_dir" ]; then \
            plugin_name=$(basename "$plugin_dir"); \
            go build -buildmode=plugin -o /app/plugins/${plugin_name}.so ./pkg/plugins/${plugin_name}/.; \
        fi \
    done

# Build the main Go application
# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/ms ./cmd/ms/.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/ms/.

# # Use a minimal alpine image for the final stage
FROM alpine:latest

# Install libc for CGO-based binary
RUN apk add --no-cache libc6-compat

# # Set the working directory inside the container
WORKDIR /app/

# Copy built binary and plugins
COPY --from=builder /app/main .
COPY --from=builder /app/plugins/*.so ./plugins/

COPY --from=builder /app/.env .

# Create plugin directory with correct permissions
RUN mkdir -p /plugins && \
    chmod 755 /plugins

VOLUME /plugins

# Expose the port the app runs on
EXPOSE 2345

# Command to run the executable
CMD ["./main"]