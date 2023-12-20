#!/bin/bash
# This script generates a CA, IA, and a Web Server Certificate.
# It stores certificates in the /ca volume. The generated ca.crt should be copied to the client.
# This script is intended for environments without a custom CA.

log() {
    echo "INFO: $@"
}

# Exit on errors or undefined variables
set -Eeuo pipefail

# Constants for certificate names
DOMAIN="nvcr.io"
PROJ_NAME="ContainerCaching"
CAID="$(hostname -f) $(date "+%Y.%m.%d %H:%M")"
CN_CA="${PROJ_NAME} CA Root ${CAID:0:32}"
CN_IA="${PROJ_NAME} Intermediate ${CAID:0:32}"
CN_WEB="${PROJ_NAME} Web Cert ${CAID:0:32}"
SAN="DNS:${DOMAIN}"

# Directory setup
# These the /ca dir should have been created by the Dockerfile and mounted as
# a volume. We specify it again so this script can be run standalone.
mkdir -p /certs /ca
cd /ca

# Default file paths
CA_KEY_FILE=${CA_KEY_FILE:-/ca/ca.key}
CA_CRT_FILE=${CA_CRT_FILE:-/ca/ca.crt}
CA_SRL_FILE=${CA_SRL_FILE:-/ca/ca.srl}

# Create or reuse CA
if [ ! -f "$CA_CRT_FILE" ]; then
    log "Generating CA cert."
    openssl genrsa -out ${CA_KEY_FILE} 4096
    openssl req -new -x509 -days 1300 -sha256 -key ${CA_KEY_FILE} -out ${CA_CRT_FILE} \
        -subj "/C=US/ST=CA/L=SF/O=ENG/OU=DEV/CN=${CN_CA}" \
        -extensions IA -config <(cat <<-EOF
[req]
distinguished_name = dn
[dn]
[IA]
basicConstraints = critical,CA:TRUE
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
subjectKeyIdentifier = hash
EOF
    )
    echo 01 > ${CA_SRL_FILE}
else
    log "Reusing existing CA cert."
fi

# Generate IA Key and Certificate
cd /certs
log "Generating IA certificate for ${DOMAIN}."
openssl genrsa -out ia.key 4096
openssl req -new -key ia.key -out ia.csr \
    -subj "/C=US/ST=CA/L=SF/O=ENG/OU=DEV/CN=${CN_IA}" \
    -reqexts IA -config <(cat <<-EOF
[req]
distinguished_name = dn
[dn]
[IA]
basicConstraints = critical,CA:TRUE,pathlen:0
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
subjectKeyIdentifier = hash
EOF
)

# Generate IA certificate cert signing request
openssl x509 -req -days 730 -in ia.csr -CA ${CA_CRT_FILE} -CAkey ${CA_KEY_FILE} -CAserial ${CA_SRL_FILE} \
    -out ia.crt -extensions IA -extfile <(cat <<-EOF
[req]
distinguished_name = dn
[dn]
[IA]
basicConstraints = critical,CA:TRUE,pathlen:0
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
subjectKeyIdentifier = hash
EOF
)

# Generate Web Server Certificate
log "Generating web certificate for ${DOMAIN}."
openssl genrsa -out cache.key 2048
openssl req -new -key cache.key -sha256 -out web.csr \
    -subj "/C=US/ST=CA/L=SF/O=ENG/OU=DEV/CN=${CN_WEB}" \
    -reqexts SAN -config <(cat <<-EOF
[req]
distinguished_name = dn
[dn]
[SAN]
subjectAltName=${SAN}
EOF
)
# Generate web certificate cert signing request
openssl x509 -req -days 365 -in web.csr -CA ia.crt -CAkey ia.key -out web.crt \
    -extensions SAN -extfile <(cat <<-EOF
[req]
distinguished_name = dn
[dn]
[SAN]
subjectAltName=${SAN}
EOF
)

# Concatenate fullchain.pem and fullchain_with_key.pem
cat web.crt ia.crt ${CA_CRT_FILE} > fullchain.pem
cat fullchain.pem cache.key > fullchain_with_key.pem
log "Certificates generated successfully."
