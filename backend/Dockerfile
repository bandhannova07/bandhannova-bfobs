# Use a lightweight Go builder image
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy the source code (Assuming we are in the backend root)
COPY . .

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o api-hunter ./cmd/api

# Final stage: Use a minimal alpine image
FROM alpine:latest

# Add certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary and .env from the builder
COPY --from=builder /app/api-hunter .
COPY --from=builder /app/.env .

# Hugging Face Spaces usually uses port 7860
ENV PORT=7860
EXPOSE 7860

# Run the application
CMD ["./api-hunter"]
