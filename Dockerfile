FROM alpine:3.18

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata \
    && cat /etc/ssl/certs/ca-cert-*.pem > /etc/ssl/certs/ca-certificates.crt

# Set the working directory to /libredesk
WORKDIR /libredesk

# Copy necessary files
COPY libredesk .
COPY config.sample.toml config.toml

# Expose port 9000 for the application
EXPOSE 9000

# Set the default command to run the libredesk binary
CMD ["./libredesk"]
