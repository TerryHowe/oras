# NVCF Container Cache

A comprehensive container caching solution optimized for NGC, S3, HuggingFace, and other container registries. This solution enhances Docker image pull efficiency by caching images locally, reducing network bandwidth and improving pull times.

## Overview

The NVCF Container Cache acts as a transparent caching proxy for container registries, providing:
- **Container Image Caching**: Local caching of container images from NGC and other registries
- **Multi-Protocol Support**: Caching for S3, NGC, HuggingFace, and standard container registries
- **High Performance**: Optimized nginx-based proxy with intelligent buffering and caching strategies
- **Security**: Optional Vault integration for certificate management and authentication
- **Scalability**: Support for both persistent and ephemeral storage configurations

## Prerequisites

1. **Container Image Access**: Access to the container cache image at `nvcr.io/nvstaging/clara/nvcf-container-caching`
2. **Kubernetes Cluster**: Running Kubernetes with containerd runtime
3. **NGC API Key**: Valid NGC API key for image pull access
4. **Storage**: Either persistent volumes or sufficient node storage for caching

## Quick Start

### 1. Create Namespace and Pull Secret

```bash
# Create dedicated namespace
kubectl create namespace container-caching

# Create NGC pull secret
kubectl create secret docker-registry ngc-container-pull \
  --docker-server=nvcr.io \
  --docker-username='$oauthtoken' \
  --docker-password=<your-ngc-key> \
  -n container-caching
```

### 2. Basic Installation (Without Vault, With Persistent Storage)

```bash
# Clone and navigate to chart directory
cd deploy/

# Install with default configuration
helm install nvcf-container-cache . \
  --namespace container-caching \
  --set vault.enabled=false
```

### 3. Verify Installation

```bash
kubectl --namespace=container-caching get pod,daemonset,svc,statefulset,pvc \
  --selector='app.kubernetes.io/name=nvcf-container-cache'
```

## Configuration Guide

### Storage Configuration

#### Option 1: Persistent Volume Storage (Recommended for Production)

```yaml
# values.yaml
persistentVolumeClaim:
  # Storage class for persistent volumes
  storageClassName: "your-storage-class"  # e.g., "azurefile", "gp2", "standard"
  
  # Size for container image cache
  sizeGB: 100
  
  # Size for proxy cache (S3/NGC/HF)
  sizeProxyGB: 200
  
  # Minimum free space percentage to maintain
  freeProxyPct: 7
```

**Example for Different Cloud Providers:**

```yaml
# Azure
persistentVolumeClaim:
  storageClassName: "azurefile"
  sizeGB: 500
  sizeProxyGB: 1000

# AWS
persistentVolumeClaim:
  storageClassName: "gp2"
  sizeGB: 500
  sizeProxyGB: 1000

# GCP
persistentVolumeClaim:
  storageClassName: "standard"
  sizeGB: 500
  sizeProxyGB: 1000
```

#### Option 2: EmptyDir Storage (For Testing/Development)

```yaml
# values.yaml
persistentVolumeClaim:
  storageClassName: "emptydir"
  sizeGB: 50
  sizeProxyGB: 100
```

**⚠️ Note**: EmptyDir storage is ephemeral and will be lost when pods restart.

### Vault Integration

#### Option 1: With Vault (Recommended for Production)

```yaml
# values.yaml
vault:
  enabled: true
  namespace: "nvcf"
  
  # Cluster configuration
  clusterCSP: "azure"  # or "gcp", "aws", "dgxc"
  clusterRegion: "eastus"
  clusterAccountName: "your-account"
  clusterName: "your-cluster-name"
  
  # Vault server configuration
  vaultAddress: "https://vault.your-domain.com"
  certLocation: "http://crls.your-domain.com/ca/pem"
```

**Example Configurations by Environment:**

```yaml
# Production
vault:
  enabled: true
  namespace: "nvcf"
  clusterCSP: "azure"
  clusterRegion: "eastus"
  clusterAccountName: "prod-account"
  clusterName: "prod-cluster"
  vaultAddress: "https://vault.nvidia.com"

# Staging
vault:
  enabled: true
  namespace: "nvcf"
  clusterCSP: "azure"
  clusterRegion: "eastus"
  clusterAccountName: "stage-account"
  clusterName: "stage-cluster"
  vaultAddress: "https://stg.vault.nvidia.com:443"
```

#### Option 2: Without Vault (Self-Signed Certificates)

```yaml
# values.yaml
vault:
  enabled: false
```

When Vault is disabled, the system automatically generates self-signed certificates for TLS communication.

### Node Selection and Scheduling

#### Node Selector Configuration

```yaml
# values.yaml
nodeSelector:
  # Target specific node groups
  nodeGroup: "monitoring"
  
  # Target nodes with specific instance types
  node.kubernetes.io/instance-type: "m5.xlarge"
  
  # Target nodes in specific zones
  topology.kubernetes.io/zone: "us-west-2a"
  
  # Custom labels
  cache-enabled: "true"
```

#### Multi-Node Deployment with Replica Configuration

```yaml
# values.yaml
replicaCount: 3  # Number of cache replicas

# Advanced affinity rules for better distribution
# Note: The chart includes anti-affinity by default to spread replicas across nodes
```

### Cache Configuration

#### Basic Cache Settings

```yaml
# values.yaml
cache:
  # Cache key storage size
  keyStorageSize: "50m"  # 10m, 50m, 100m
  
  # Maximum cache size
  maxSize: "500g"  # 80g, 500g, 1000g
  
  # Inactive period (how long items stay in cache without access)
  inactive: "7d"  # 1d, 7d, 30d
  
  # Cache validity period
  valid: "24h"  # 4h, 24h, 7d
  
  # HTTP/2 support
  http2: "on"  # on/off
  
  # Worker connections
  workerConnection: 2000  # 1000, 2000, 4000
```

#### Advanced Cache Tuning

```yaml
# values.yaml
cache:
  # For high-traffic environments
  keyStorageSize: "200m"
  maxSize: "2000g"
  workerConnection: 4000
  
  # For development environments
  keyStorageSize: "10m"
  maxSize: "100g"
  workerConnection: 1000
```

### Network and Service Configuration

#### Service Type Options

```yaml
# values.yaml
service:
  # ClusterIP (internal access only)
  type: ClusterIP
  port: 14128
  
  # NodePort (external access via node ports)
  type: NodePort
  port: 30345
  
  # LoadBalancer (cloud provider load balancer)
  type: LoadBalancer
  port: 14128
```

#### Target Host Configuration

```yaml
# values.yaml
# Single target
targetHost: "nvcr.io"

# Multiple targets
targetHost: "nvcr.io,gcr.io,docker.io"

# Environment-specific targets
targetHost: "stg.nvcr.io,nvcr.io"
```

### Container Configuration

#### Image Configuration

```yaml
# values.yaml
images:
  # Main cache server
  server: "nvcr.io/nv-ngc-devops/nvcf-container-cache:v1.1.32"
  
  # Prometheus exporter
  exporter: "nvcr.io/nv-ngc-devops/nginx-prometheus-exporter:1.0"
  
  # Certificate management
  certificates: "nvcr.io/nv-ngc-devops/nvcf-proxy-tls-certs:1.2.1"
  
  # Pull secrets
  secrets:
    - "ngc-container-pull"
    - "additional-pull-secret"
```

### Monitoring and Metrics

#### Enable Monitoring

```yaml
# values.yaml
monitoring:
  enabled: true

metrics:
  # Metrics storage size
  cacheMetricsStorageSize: "300m"
  
  # Throughput histogram buckets (bytes/sec)
  throughputHistogramBuckets: "25000000, 30000000, 35000000, 40000000, 50000000, 60000000, 80000000, 100000000"
```

#### OpenTelemetry Tracing

```yaml
# values.yaml
traces:
  enabled: true
  collector:
    endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
```

### Feature-Specific Configuration

#### Nucleus Integration

```yaml
# values.yaml
nucleus:
  enabled: true  # Enable for NVCF nucleus integration
```

## Installation Examples

### Example 1: Production with Vault and Persistent Storage

```yaml
# production-values.yaml
replicaCount: 3

nodeSelector:
  nodeGroup: "cache"
  node.kubernetes.io/instance-type: "m5.2xlarge"

vault:
  enabled: true
  namespace: "nvcf"
  clusterCSP: "azure"
  clusterRegion: "eastus"
  clusterAccountName: "prod-account"
  clusterName: "prod-cluster-east"
  vaultAddress: "https://vault.nvidia.com"

persistentVolumeClaim:
  storageClassName: "premium-ssd"
  sizeGB: 1000
  sizeProxyGB: 2000

cache:
  keyStorageSize: "200m"
  maxSize: "2000g"
  inactive: "30d"
  valid: "7d"
  workerConnection: 4000

service:
  type: LoadBalancer
  port: 14128

monitoring:
  enabled: true

traces:
  enabled: true
```

```bash
helm install nvcf-container-cache deploy/ \
  --namespace container-caching \
  --values production-values.yaml
```

### Example 2: Development without Vault, EmptyDir Storage

```yaml
# development-values.yaml
replicaCount: 1

nodeSelector:
  nodeGroup: "dev"

vault:
  enabled: false

persistentVolumeClaim:
  storageClassName: "emptydir"
  sizeGB: 50
  sizeProxyGB: 100

cache:
  keyStorageSize: "10m"
  maxSize: "100g"
  inactive: "1d"
  valid: "4h"
  workerConnection: 1000

service:
  type: NodePort
  port: 30345

monitoring:
  enabled: false
```

```bash
helm install nvcf-container-cache-dev deploy/ \
  --namespace container-caching \
  --values development-values.yaml
```

### Example 3: High-Performance Multi-Cloud Setup

```yaml
# multi-cloud-values.yaml
replicaCount: 5

nodeSelector:
  cache-tier: "high-performance"

vault:
  enabled: true
  namespace: "nvcf"
  clusterCSP: "gcp"
  clusterRegion: "us-central1"
  clusterAccountName: "multi-cloud-account"
  clusterName: "gcp-central1-prod"
  vaultAddress: "https://vault.nvidia.com"

persistentVolumeClaim:
  storageClassName: "ssd-retain"
  sizeGB: 2000
  sizeProxyGB: 5000
  freeProxyPct: 10

cache:
  keyStorageSize: "500m"
  maxSize: "4000g"
  inactive: "60d"
  valid: "14d"
  http2: "on"
  workerConnection: 8000

targetHost: "nvcr.io,gcr.io,us-docker.pkg.dev"

service:
  type: LoadBalancer
  port: 14128

metrics:
  cacheMetricsStorageSize: "1g"
  throughputHistogramBuckets: "50000000, 75000000, 100000000, 150000000, 200000000"

traces:
  enabled: true
  collector:
    endpoint: "jaeger-collector.observability.svc.cluster.local:14250"

nucleus:
  enabled: true

monitoring:
  enabled: true
```

```bash
helm install nvcf-container-cache-hpc deploy/ \
  --namespace container-caching \
  --values multi-cloud-values.yaml
```

## Post-Installation Configuration

### 1. Verify DaemonSet Deployment

The installation includes a DaemonSet that configures containerd on each node:

```bash
# Check DaemonSet status
kubectl get daemonset -n container-caching

# Check configuration on nodes
kubectl logs -n container-caching daemonset/nvcf-container-cache-cc
```

### 2. Test Cache Functionality

```bash
# Test with a sample image pull
docker pull your-cache-service:30345/nvidia/cuda:11.8-base-ubuntu22.04

# Check cache metrics
kubectl port-forward -n container-caching svc/nvcf-container-cache 9113:9113
curl http://localhost:9113/metrics | grep container_cache
```

### 3. Monitor Performance

```bash
# Check cache hit ratio
kubectl logs -n container-caching statefulset/nvcf-container-cache | grep "cache.*HIT"

# Monitor storage usage
kubectl exec -n container-caching nvcf-container-cache-0 -- df -h /container_cache /proxy_cache
```

## Troubleshooting

### Common Issues

#### 1. Storage Issues

```bash
# Check PVC status
kubectl get pvc -n container-caching

# Check storage class availability
kubectl get storageclass

# For EmptyDir issues, check node disk space
kubectl describe nodes | grep -A5 -B5 "disk pressure"
```

#### 2. Vault Connection Issues

```bash
# Check vault configuration
kubectl get configmap -n container-caching nvcf-container-cache-vault-config -o yaml

# Check service account tokens
kubectl describe serviceaccount -n container-caching

# Test vault connectivity
kubectl exec -n container-caching nvcf-container-cache-0 -- curl -k https://vault.nvidia.com/v1/sys/health
```

#### 3. DaemonSet Configuration Issues

```bash
# Check DaemonSet logs
kubectl logs -n container-caching daemonset/nvcf-container-cache-cc

# Verify containerd configuration
kubectl exec -n container-caching ds/nvcf-container-cache-cc -- cat /host/etc/containerd/config.toml
```

#### 4. Performance Issues

```bash
# Check resource usage
kubectl top pods -n container-caching

# Monitor nginx error logs
kubectl logs -n container-caching nvcf-container-cache-0 -c server | grep ERROR

# Check cache statistics
kubectl exec -n container-caching nvcf-container-cache-0 -- wget -qO- http://localhost:13128/stub_status
```

## Uninstallation

### 1. Remove Helm Deployment

```bash
helm uninstall nvcf-container-cache -n container-caching
```

### 2. Clean Up Node Configuration

```bash
# Deploy cleanup DaemonSet
kubectl apply -f client/remove/remove-containerd-configuration.yaml

# Wait for completion
kubectl wait --for=condition=complete job/remove-containerd-config --timeout=300s

# Remove cleanup resources
kubectl delete -f client/remove/remove-containerd-configuration.yaml
```

### 3. Remove Namespace and PVCs

```bash
# Delete PVCs (if needed)
kubectl delete pvc -n container-caching --all

# Delete namespace
kubectl delete namespace container-caching
```

## Advanced Configuration

### Custom nginx Configuration

You can customize nginx configuration by modifying the configuration files in `deploy/files/`:

- `nginx.conf`: Main nginx configuration
- `proxy-cache.conf`: S3 caching configuration
- `container-cache.conf`: Container registry caching
- `proxy-common.conf`: Common proxy settings

### SSL/TLS Configuration

For custom SSL configurations without Vault:

```yaml
# Custom certificate configuration
# Place certificates in a ConfigMap or Secret and mount them
```

### Performance Tuning

For high-throughput environments:

```yaml
cache:
  workerConnection: 16384
  keyStorageSize: "1g"
  maxSize: "10000g"

# Consider node-specific optimizations
nodeSelector:
  node.kubernetes.io/instance-type: "c5n.4xlarge"  # High network performance
```

## Support and Documentation

- **Configuration Files**: See `deploy/files/` for nginx configuration details
- **Templates**: See `deploy/templates/` for Kubernetes resource templates
- **Examples**: See `deploy/values-*.yaml` for environment-specific examples
- **Monitoring**: Prometheus metrics available at `/metrics` endpoint

For additional support, refer to the NVIDIA NGC documentation and your internal DevOps team.