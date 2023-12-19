# Container Caching


### Overview

This project provides a container caching solution specifically optimized for nvcr.io. It is designed to enhance the efficiency of Docker image pulls from nvcr.io by caching the images locally, reducing network bandwidth and improving pull times for frequently accessed images.

### Setup Instructions

#### Kubernetes (via Microk8s)
See [kubernetes](./kubernetes/) for a sample deployment (under construction)


1. Accessing MicroK8s containerd Configuration

MicroK8s stores its containerd configuration in a different location due to its snap-based nature. The configuration file is typically located at /var/snap/microk8s/current/args/containerd-template.toml.

    Edit the containerd-template.toml file:
`sudo vi /var/snap/microk8s/current/args/containerd-template.toml`

2. Add the registry mirror config for nvcr.io
```toml
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."nvcr.io"]
endpoint = ["https://container-cache:13128"]  # Replace with your Nginx proxy address

```

3. Adding a Custom SSL Certificate

You need to add the certificate to MicroK8s’ trusted certificates.
   Place your custom certificate (e.g., the one in `certs_dir/ca.crt`) in the MicroK8s certificates directory:

`sudo cp /path/to/ca.crt /var/snap/microk8s/current/certs/``

Update the certificate authority (CA) certificates:
```bash
sudo microk8s.refresh-certs

```
Restart MicroK8s

```bash
sudo microk8s.stop
sudo microk8s.start

```

#### Using Docker:
  To start the caching proxy, run the following command:


#### Testing the Configuration

To validate the setup:

    Pull an image from nvcr.io to see if it goes through the cache.
    Check the logs from the cache server to verify that the requests are being routed correctly.

```bash
    # build the base image 
    pushd base_image
    docker build -t caching-base:latest -f Dockerfile-base . --no-cache
    popd

    # build the image
    docker build -t cache-server:latest -f Dockerfile . --no-cache
    docker run -d --name container-cache -p 13128:13128 container-cache
```

#### Running the Solution locally (development)
Using Docker Compose:

A docker-compose.yml file is provided for easy setup. To use it, run:

```bash
docker-compose up -d
```


Configuring Docker to Use the Cache

```bash
mkdir -p /etc/systemd/system/docker.service.d
# Assuming you are running locally, change if running the cache on a different maching
sudo vim /etc/systemd/system/docker.service.d/http-proxy.conf

# Add the following to /etc/systemd/system/docker.service.d/http-proxy.conf:
[Service]
Environment="HTTP_PROXY=http://localhost:13128/"
Environment="HTTPS_PROXY=http://localhost:13128/"

# Once you start the server (with an initial docker-compose up)
cat certs_dir/ca.crt > /usr/share/ca-certificates/caching.crt
echo "caching.crt" >> /etc/ca-certificates.conf
update-ca-certificates --fresh
sudo systemctl daemon-reload
sudo systemctl restart docker
```
Note: Try to keep the certs_dir around, you need to run the above command every time they are generated.

### Additional Information

For more detailed information about the configuration and customization options, please refer to the provided configuration files and scripts in this repository.