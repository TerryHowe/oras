#! /bin/bash

log() {
    echo "INFO: $@"
}

log "Entrypoint starting."

# Setup error handling
set -Eeuo pipefail
trap "echo TRAPed signal" HUP INT QUIT TERM

# Extract the first IPv4 nameserver address from /etc/resolv.conf and
# set it as the resolver in /etc/nginx/resolvers.conf with IPv6 disabled.

# Path to the Nginx resolver configuration file
NGINXRESOLVER="/etc/nginx/resolver.conf"

# Extract the first nameserver and validate it's an IPv4 address
NAMESERVER=$(awk '/^nameserver/{print $2; exit}' /etc/resolv.conf | grep -Eo '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$')
# Check if a valid IPv4 nameserver was found
if [ -z "${NAMESERVER}" ]; then
    echo "Error: No valid IPv4 nameserver found in /etc/resolv.conf" >&2
    exit 1
fi
# Write the resolver configuration to the Nginx configuration file
echo "resolver ${NAMESERVER} ipv6=off;" > "${NGINXRESOLVER}"
# Confirm success
echo "Successfully set resolver to ${NAMESERVER} in ${NGINXRESOLVER}"

# Generate certificates for nvcr.io
log "Generating certificates for nvcr.io"
/generate_certs.sh

# Validate the NGINX configuration
log "Validating NGINX Configuration."
nginx -t

log "Container Cache is running."
nginx -g "daemon off;"
