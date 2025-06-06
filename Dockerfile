FROM alpine:3.19

# Set working directory
WORKDIR /app

# Copy prebuilt binaries
COPY github-fetch .
COPY test-binaries/ /app/test-binaries/

COPY .env /app/.env

# Set executable permission (if needed)
RUN chmod +x github-fetch
RUN chmod +x /app/test-binaries/*

# Default command
CMD ["./github-fetch"]