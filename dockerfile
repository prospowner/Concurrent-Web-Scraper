# --- STAGE 1: Build the binary ---
FROM golang:alpine AS builder

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod ./
COPY go.sum ./

# Copy the source code
COPY . .

# Compile the go app into a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# --- STAGE 2: Minimal runtime image ---
FROM alpine:latest

# Install CA certificates so HTTPS/API requests don't fail
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Set the default entrypoint for the container
ENTRYPOINT ["./main"]