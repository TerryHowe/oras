# Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
#
# NVIDIA CORPORATION and its licensors retain all intellectual property
# and proprietary rights in and to this software, related documentation
# and any modifications thereto.  Any use, reproduction, disclosure or
# distribution of this software and related documentation without an express
# license agreement from NVIDIA CORPORATION is strictly prohibited.

# Use ARGs to define versions
ARG OPENRESTY_VERSION=openresty-1.21.4.3

#Use specific version of ubuntu
FROM nvcr.io/nvstaging/clara/ubuntu-jammy-amd64:current as builder

# Re-define ARGs inside the build stage
ARG OPENRESTY_VERSION

# Avoid APT warnings
ARG DEBIAN_FRONTEND=noninteractive

# Create appuser.
ENV USER=nvs
ENV UID=1000

RUN useradd \
    --home-dir "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

# Install dependencies
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y \
    build-essential \
        curl \
        libpcre3 \
        libpcre3-dev \
        libcrypt1 \
        libgcc1 \
        libssl-dev \
        libz1 \
        openssl \
        wget \
        zlib1g \
        zlib1g-dev       

# Create the cache directory and CA directories
RUN mkdir -p /container_cache /var/cache/nginx && touch /var/run/nginx.pid

# Set working directory
WORKDIR /usr/src

# Download and extract OpenResty
RUN wget https://openresty.org/download/${OPENRESTY_VERSION}.tar.gz && \
    tar -zxvf ${OPENRESTY_VERSION}.tar.gz

# Custom NGINX configuration
ARG NGINX_CONFIG="\
    # User and group settings
    --user=nvs \
    --group=nvs \
    --conf-path=/etc/nginx/nginx.conf \
    --error-log-path=/var/log/nginx/error.log \
    --http-log-path=/var/log/nginx/access.log \
    --pid-path=/var/run/nginx.pid \
    --lock-path=/var/run/nginx.lock \
    # Temporary path settings
    --http-client-body-temp-path=/var/cache/nginx/client_temp \
    --http-proxy-temp-path=/var/cache/nginx/proxy_temp \
    --http-fastcgi-temp-path=/var/cache/nginx/fastcgi_temp \
    --http-uwsgi-temp-path=/var/cache/nginx/uwsgi_temp \
    --http-scgi-temp-path=/var/cache/nginx/scgi_temp \
    # Modules configuration
    --with-compat \
    --with-file-aio \
    --with-http_addition_module \
    --with-http_auth_request_module \
    --with-http_gunzip_module \
    --with-http_gzip_static_module \
    --with-http_random_index_module \
    --with-http_realip_module \
    --with-http_secure_link_module \
    --with-http_slice_module \
    --with-http_ssl_module \
    --with-http_stub_status_module \
    --with-http_sub_module \
    --with-http_v2_module \
    --with-threads \
    --with-stream \
    --with-stream_realip_module \
    --with-stream_ssl_module \
    --with-stream_ssl_preread_module \
    "

# Compile and install OpenResty
WORKDIR ${OPENRESTY_VERSION}
RUN ./configure ${NGINX_CONFIG} &&\
    make && make install \
    && ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log

# Set PATH for openresty
ENV PATH=/usr/local/openresty/bin:/usr/local/openresty/nginx/sbin:/usr/local/openresty/luajit/bin/:$PATH

# Add needed Lua modules
RUN opm get ledgetech/lua-resty-http=0.17.1 && opm get knyar/nginx-lua-prometheus=0.20230607

# Run from the root directory (unset initial change to root directory)
WORKDIR /

# Sanity check of binary and default nginx.conf
RUN nginx -v && nginx -t

# Define default command
CMD ["bash"]

# Distroless image
FROM nvcr.io/nvidian/distroless/go:v3.0.5

# Copy files to distroless container
COPY --from=builder /etc/nginx /etc/nginx
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --chown=nvs:nvs --from=builder /var/cache /var/cache
COPY --chown=nvs:nvs --from=builder /var/run /var/run
COPY --chown=nvs:nvs --from=builder /var/log /var/log
COPY --chown=nvs:nvs --from=builder /container_cache /container_cache
COPY --from=builder /usr/local/openresty /usr/local/openresty
COPY --from=builder /lib/x86_64-linux-gnu/libcrypt.so.1 /lib/x86_64-linux-gnu/ 
COPY --from=builder /lib/x86_64-linux-gnu/libpcre.so.3 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libz.so.1 /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libgcc_s.so.1 /lib/x86_64-linux-gnu/

# Expose Cache as a volume for persistence
VOLUME /container_cache

# Set PATH for openresty
ENV PATH=/usr/local/openresty/bin:/usr/local/openresty/nginx/sbin:/usr/local/openresty/luajit/bin:$PATH

USER nvs:nvs

CMD ["nginx", "-g", "daemon off;"]
