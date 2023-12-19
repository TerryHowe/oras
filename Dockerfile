ARG FROM_IMAGE=caching-base:latest

# Base image has NGINX build from Source with Lua and needed modules
FROM ${FROM_IMAGE}

# Avoid APT warnings
ARG DEBIAN_FRONTEND=noninteractive

RUN apt update
# Add necessary packages
RUN apt install bash ca-certificates openssl

# Create the cache directory and CA directories
RUN mkdir -p /container_cache /ca

# Expose Cache as a volume for persistence
VOLUME /container_cache

# Expose /ca as a volume. 
# Prevents us from needing to update-ca-certificates on the clients every time we add a new cert.
VOLUME /ca

# Add NGINX configs
ADD nginx.conf /etc/nginx/nginx.conf

# Add and make executable the entrypoint script and script to create the CA certificate
ADD startup.sh /startup.sh
ADD generate_certs.sh /generate_certs.sh
RUN chmod +x /generate_certs.sh /startup.sh

# Clients should only use 13128, not anything else.
EXPOSE 13128

# Run the entrypoint script
ENTRYPOINT ["/startup.sh"]
