# NVCF Container Cache - Comprehensive Configuration and Deployment Guide

**Version:** 0.25.9  
**Application Version:** 1.2.1  
**Chart Name:** nvcf-container-cache

## Table of Contents
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Configuration Reference](#configuration-reference)
- [Deployment Guide](#deployment-guide)
- [Testing Guide](#testing-guide)
- [Environment-Specific Examples](#environment-specific-examples)
- [Troubleshooting](#troubleshooting)
- [Advanced Configuration](#advanced-configuration)

---

## Overview

The NVCF Container Cache is a high-performance caching proxy optimized for container registries, particularly NGC (NVIDIA GPU Cloud). It acts as a transparent proxy that caches container images, layers, and metadata to reduce bandwidth usage and improve image pull times across Kubernetes clusters.

### Key Features
- **Container Registry Caching**: Supports NGC, Docker Hub, GCR, ECR, and other OCI-compliant registries
- **Multi-Protocol Support**: Container images, S3 objects, HuggingFace models, and NGC assets
- **High Performance**: Nginx-based proxy with intelligent caching and buffering
- **Security**: Vault integration for certificate management or self-signed certificates
- **Scalability**: Multi-replica deployment with persistent or ephemeral storage
- **Observability**: Prometheus metrics and detailed logging

---

## Prerequisites

1. **Kubernetes Cluster**: 1.19+ with containerd runtime
2. **Container Image Access**: Access to the public NVCF container cache images from [NGC Catalog](https://catalog.ngc.nvidia.com/orgs/nvidia/teams/nvcf-byoc/helm-charts/nvcf-container-cache)
3. **NGC API Key**: Valid NGC API key for accessing NVIDIA container images
4. **Storage**: Persistent volumes or sufficient node storage for caching
5. **Network**: Outbound HTTPS access to container registries
6. **Tools**: kubectl, helm 3.x

---

## Configuration Reference

All configuration is managed through the Helm chart's `values.yaml` file. Below is the complete reference of all configurable options.

### Core Configuration

#### Node Selection and Replicas

```yaml
# Node selection for cache pods
nodeSelector:
  nodeGroup: monitoring                           # Target specific node groups
  node.kubernetes.io/instance-type: "m5.xlarge"  # Specific instance types
  topology.kubernetes.io/zone: "us-west-2a"      # Specific availability zones
  cache-enabled: "true"                          # Custom node labels

# Number of cache replicas (default: 1)
replicaCount: 3
```

**Options:**
- `nodeSelector`: Kubernetes node selector labels to target specific nodes
- `replicaCount`: Number of cache replicas (1-10, typically 3-5 for production)

#### Target Registries

```yaml
# Target registries for container caching (default: nvcr.io)
targetHost: "nvcr.io"                    # Single registry
targetHost: "nvcr.io,gcr.io,docker.io"  # Multiple registries
targetHost: "stg.nvcr.io,nvcr.io"       # Staging and production
```

**Options:**
- Single or comma-separated list of container registries
- Common values: `nvcr.io`, `stg.nvcr.io`, `gcr.io`, `docker.io`, `registry-1.docker.io`

### Container Images

```yaml
images:
  # Main cache server image
  server: nvcr.io/nvidia/nvcf-byoc/nvcf-container-cache:latest
  
  # Prometheus metrics exporter
  exporter: nvcr.io/nvidia/nvcf-byoc/nginx-prometheus-exporter:latest
  
  # TLS certificate bundle
  certificates: nvcr.io/nvidia/nvcf-byoc/nvcf-proxy-tls-certs:latest
  
  # Pull secrets for image access
  secrets:
    - nvidia-ngcuser-pull-secret     # NGC pull secret
    - additional-pull-secret         # Additional secrets if needed
```

### Cache Configuration

```yaml
cache:
  # Cache key storage size (default: 10m)
  keyStorageSize: "50m"      # Options: 10m, 50m, 100m, 500m
  
  # Maximum cache size (default: 80g)
  maxSize: "500g"            # Options: 100g, 500g, 1000g, 5000g
  
  # Inactive period before removal (default: 60d)
  inactive: "7d"             # Options: 1d, 7d, 30d, 60d
  
  # Cache validity period (default: 24h)
  valid: "12h"               # Options: 4h, 12h, 24h, 7d
  
  # HTTP/2 support (default: off)
  http2: "on"                # Options: on, off
  
  # Worker connections (default: 1000)
  workerConnection: 2000     # Options: 1000, 2000, 4000, 8000
```

**Cache Size Guidelines:**
- **Development**: keyStorageSize: 10m, maxSize: 100g
- **Staging**: keyStorageSize: 50m, maxSize: 500g  
- **Production**: keyStorageSize: 200m, maxSize: 2000g+

### Storage Configuration

#### Persistent Volume Claims

```yaml
persistentVolumeClaim:
  # Storage class name
  storageClassName: "gp2"           # AWS: gp2, gp3
  storageClassName: "azurefile"     # Azure: azurefile, premium-ssd
  storageClassName: "standard"      # GCP: standard, ssd-retain
  storageClassName: "emptydir"      # EmptyDir (ephemeral, for testing)
  
  # Container cache volume size (default: 100)
  sizeGB: 500                       # Size in GB for container images
  
  # Proxy cache volume size (default: 200)  
  sizeProxyGB: 1000                 # Size in GB for S3/NGC/HF caching
  
  # Minimum free space percentage (default: 7)
  freeProxyPct: 10                  # Cleanup when reaching 90% full
```

**Storage Class Examples:**

| Cloud Provider | Storage Class | Type | Performance |
|----------------|---------------|------|-------------|
| AWS | `gp2` | General Purpose SSD | Standard |
| AWS | `gp3` | General Purpose SSD | Enhanced |
| AWS | `io1` | Provisioned IOPS | High Performance |
| Azure | `azurefile` | File Storage | Standard |
| Azure | `premium-ssd` | Premium SSD | High Performance |
| GCP | `standard` | Persistent Disk | Standard |
| GCP | `ssd-retain` | SSD | High Performance |
| Any | `emptydir` | EmptyDir | Ephemeral (testing only) |

### Service Configuration

```yaml
service:
  # Service type (default: ClusterIP)
  type: ClusterIP              # Internal cluster access only
  type: NodePort               # External access via node ports
  type: LoadBalancer           # Cloud provider load balancer
  
  # Service port (default: 14128)
  port: 14128                  # ClusterIP/LoadBalancer port
  port: 30345                  # NodePort (30000-32767 range)
```

**Service Type Guidelines:**
- **ClusterIP**: Internal cluster access (most common)
- **NodePort**: Direct node access for testing
- **LoadBalancer**: Production external access

### Resource Configuration

```yaml
resources:
  requests:
    memory: "8Gi"              # Minimum memory
    cpu: "2"                   # Minimum CPU cores
  limits:
    memory: "32Gi"             # Maximum memory
    cpu: "16"                  # Maximum CPU cores
```

**Resource Sizing Guidelines:**

| Environment | Memory Request | Memory Limit | CPU Request | CPU Limit |
|-------------|----------------|--------------|-------------|-----------|
| Development | 4Gi | 8Gi | 1 | 4 |
| Staging | 8Gi | 32Gi | 2 | 16 |
| Production | 16Gi | 64Gi | 4 | 32 |
| High-Performance | 32Gi | 128Gi | 8 | 64 |

### Vault Integration

#### Enable Vault (Production)

```yaml
vault:
  enabled: true
  namespace: "nvcf"                    # Vault namespace
  
  # Cluster identification
  clusterCSP: "azure"                  # Cloud provider: azure, gcp, aws, dgxc
  clusterRegion: "eastus"              # Region identifier
  clusterAccountName: "prod-account"   # Account/subscription name
  clusterName: "prod-cluster-east"     # Cluster identifier
  
  # Vault server details
  vaultAddress: "https://vault.nvidia.com"                    # Production
  vaultAddress: "https://stg.vault.nvidia.com:443"           # Staging
  certLocation: "http://crls.vpki.nvidia.com/ca/pem"         # Certificate location
```

#### Disable Vault (Self-Signed Certificates)

```yaml
vault:
  enabled: false
```

When Vault is disabled, the system automatically generates self-signed certificates for TLS communication.

### Metrics and Monitoring

#### Enable Monitoring

```yaml
monitoring:
  enabled: true                # Enable PodMonitor for Prometheus

metrics:
  # Metrics storage size (default: 10m)
  cacheMetricsStorageSize: "300m"
  
  # Throughput histogram buckets for performance monitoring (bytes/sec)
  throughputHistogramBuckets: "25000000, 30000000, 35000000, 40000000, 50000000, 60000000, 80000000, 100000000"
```

#### Disable Monitoring (Resource Optimization)

```yaml
monitoring:
  enabled: false
```

**Benefits of Disabling Monitoring:**
- Reduces memory usage (eliminates nginx-prometheus-exporter sidecar)
- Lower CPU overhead (no metrics collection)
- Simplified deployment (fewer Kubernetes resources)
- Reduced network traffic (no metrics scraping)

### Advanced Features

#### Nucleus Integration

```yaml
nucleus:
  enabled: true               # Enable for NVCF nucleus integration
  enabled: false              # Disable if not using unbound-dns
```

#### Tracing (OpenTelemetry)

```yaml
traces:
  enabled: false              # Enable distributed tracing
  collector:
    endpoint: "prod.otel.kaizen.nvidia.com:8282"
```

#### Custom Pod Configuration

```yaml
# Custom pod annotations
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9113"

# Custom pod labels  
podLabels:
  environment: "production"
  team: "infrastructure"
```

---

## Deployment Guide

### 1. Prepare Environment

#### Create Namespace and Pull Secret

```bash
# Create dedicated namespace
kubectl create namespace container-caching

# Create NGC pull secret (replace <your-ngc-key> with your actual API key)
kubectl create secret docker-registry nvidia-ngcuser-pull-secret \
  --docker-server=nvcr.io \
  --docker-username='$oauthtoken' \
  --docker-password=<your-ngc-key> \
  -n container-caching
```

#### Verify Prerequisites

```bash
# Check cluster access
kubectl cluster-info

# Check storage classes
kubectl get storageclass

# Verify namespace
kubectl get namespace container-caching
```

### 2. Configure Values File

Create a custom values file based on your environment:

```bash
# Copy example values file
cp deploy/values-gcp-ct1.yaml my-values.yaml

# Edit configuration
vim my-values.yaml
```

### 3. Deploy with Helm

#### Basic Deployment

```bash
# Navigate to chart directory
cd /path/to/nvcf-container-cache/deploy

# Install with default values
helm install nvcf-container-cache . \
  --namespace container-caching \
  --set vault.enabled=false
```

#### Production Deployment

```bash
# Install with custom values file
helm install nvcf-container-cache . \
  --namespace container-caching \
  --values my-values.yaml \
  --timeout 10m
```

#### Upgrade Existing Deployment

```bash
# Upgrade with new configuration
helm upgrade nvcf-container-cache . \
  --namespace container-caching \
  --values my-values.yaml \
  --timeout 10m

# Check upgrade status
helm status nvcf-container-cache -n container-caching
```

### 4. Verify Deployment

```bash
# Check all resources
kubectl -n container-caching get pods,svc,pvc,statefulset,daemonset

# Check pod status
kubectl -n container-caching get pods -l app.kubernetes.io/name=nvcf-container-cache

# Check logs
kubectl -n container-caching logs nvcf-container-cache-0 -c nginx-proxy
```

Expected output:
```
NAME                       READY   STATUS    RESTARTS   AGE
nvcf-container-cache-0     3/3     Running   0          2m
nvcf-container-cache-1     3/3     Running   0          2m
nvcf-container-cache-2     3/3     Running   0          2m
```

### 5. Configure Node Containerd

The deployment includes a DaemonSet that automatically configures containerd on each node to use the cache. Verify it's working:

```bash
# Check DaemonSet status
kubectl -n container-caching get daemonset

# Check DaemonSet logs
kubectl -n container-caching logs daemonset/nvcf-container-cache-cc

# Verify containerd configuration on nodes
kubectl -n container-caching exec ds/nvcf-container-cache-cc -- \
  cat /host/etc/containerd/config.toml | grep -A5 "registry.mirrors"
```

---

## Testing Guide

The NVCF Container Cache includes a comprehensive testing framework to validate functionality and performance.

### 1. Automated Test Suite

#### Quick Test

```bash
# Navigate to test directory
cd /path/to/nvcf-cache/tests/container-cache

# Set kubeconfig for your cluster
export KUBECONFIG=/path/to/your/kubeconfig

# Run automated test
./container-cache-test.sh
```

#### Test Commands

| Command | Description |
|---------|-------------|
| `./container-cache-test.sh` | Run full automated test suite |
| `./container-cache-test.sh --status` | Check cache pod status and recent activity |
| `./container-cache-test.sh --logs` | Stream cache logs (filtered for container traffic) |
| `./container-cache-test.sh --hits` | Show recent cache HITs/MISSes |
| `./container-cache-test.sh --cleanup` | Clean up test resources |

#### Expected Test Output

```
╔══════════════════════════════════════════════════════════════╗
║       NVCF Container Cache Test Suite for QA                 ║
╚══════════════════════════════════════════════════════════════╝

▶ Checking prerequisites...
✅ kubectl found
✅ Cluster access confirmed  
✅ Namespace 'container-caching' exists
✅ Found 3 running container cache pod(s)

✅ Found docker config at /docker-config/config.json
✅ Found registry credentials
✅ Token exchange working for stg.nvcr.io

════════════════════════════════════════════════════════════
  Testing Registry: stg.nvcr.io
════════════════════════════════════════════════════════════

═══ Test: Cache Proxy for stg.nvcr.io ═══
  Using Bearer token authentication
✅ Cache proxy working for stg.nvcr.io

═══ Test: Manifest Fetch via Cache ═══
📦 First manifest request (expecting MISS)...
  HTTP Status: 200
  ✅ First request successful (HTTP 200)

📦 Second manifest request...
  HTTP Status: 200  
  ✅ Second request completed (HTTP 200)

Tests Passed: 8
Tests Failed: 0
✅ All container cache API tests PASSED
```

### 2. Manual Testing

#### Test Cache Hit/Miss Behavior

**Terminal 1:** Watch cache logs for all pods
```bash
# Stream logs showing cache status
./container-cache-test.sh --logs

# Or manually:
kubectl -n container-caching logs -l app=nvcf-proxy-cache -c nginx-proxy -f \
  --max-log-requests=10 | grep -E '"upstream_cache_status":"(HIT|MISS)"'
```

**Terminal 2:** Pull a test image
```bash
# Pull an image through the cache
docker pull stg.nvcr.io/nvidia/cuda:11.8-base-ubuntu22.04
```

**Expected Log Output:**
```json
# First pull - MISS (fetching from upstream)
{"upstream_cache_status":"MISS", "request_uri":"/v2/nvidia/cuda/blobs/sha256:abc123..."}

# Second pull - HIT (served from cache)
{"upstream_cache_status":"HIT", "request_uri":"/v2/nvidia/cuda/blobs/sha256:abc123..."}
```

#### Performance Testing

```bash
# Test first pull (cache MISS)
time docker pull stg.nvcr.io/nvidia/cuda:11.8-base-ubuntu22.04

# Remove local image
docker rmi stg.nvcr.io/nvidia/cuda:11.8-base-ubuntu22.04

# Test second pull (cache HIT - should be faster)
time docker pull stg.nvcr.io/nvidia/cuda:11.8-base-ubuntu22.04
```

#### Check Cache Statistics

```bash
# View cache statistics
kubectl -n container-caching exec nvcf-container-cache-0 -c nginx-proxy -- \
  wget -qO- http://localhost:13128/stub_status

# Check storage usage
kubectl -n container-caching exec nvcf-container-cache-0 -- \
  df -h /container_cache /proxy_cache

# View recent cache activity
./container-cache-test.sh --hits
```

### 3. Validation Checklist

✅ **Pod Status**: All cache pods running and ready  
✅ **Service Connectivity**: Cache service accessible from cluster  
✅ **DaemonSet Config**: Containerd configured on all nodes  
✅ **Registry Connectivity**: Can reach target registries through cache  
✅ **Authentication**: NGC OAuth token exchange working  
✅ **Cache Behavior**: Observing HIT/MISS patterns in logs  
✅ **Performance**: Improved pull times for cached images  
✅ **Storage**: PVCs bound and storage available  

---

## Environment-Specific Examples

### Development Environment

```yaml
# development-values.yaml
replicaCount: 1

nodeSelector:
  nodeGroup: dev

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

resources:
  requests:
    memory: "4Gi"
    cpu: "1"
  limits:
    memory: "8Gi" 
    cpu: "4"

monitoring:
  enabled: false

service:
  type: NodePort
  port: 30345
```

Deploy:
```bash
helm install nvcf-container-cache-dev deploy/ \
  --namespace container-caching \
  --values development-values.yaml
```

### Production Environment

```yaml
# production-values.yaml
replicaCount: 3

nodeSelector:
  nodeGroup: cache
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
  freeProxyPct: 10

cache:
  keyStorageSize: "200m"
  maxSize: "2000g"
  inactive: "30d"
  valid: "7d"
  http2: "on"
  workerConnection: 4000

resources:
  requests:
    memory: "16Gi"
    cpu: "4"
  limits:
    memory: "64Gi"
    cpu: "32"

service:
  type: LoadBalancer
  port: 14128

monitoring:
  enabled: true

metrics:
  cacheMetricsStorageSize: "1g"
  throughputHistogramBuckets: "50000000, 75000000, 100000000, 150000000, 200000000"

nucleus:
  enabled: true
```

Deploy:
```bash
helm install nvcf-container-cache deploy/ \
  --namespace container-caching \
  --values production-values.yaml
```

### Multi-Cloud High-Performance

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

resources:
  requests:
    memory: "32Gi"
    cpu: "8"
  limits:
    memory: "128Gi"
    cpu: "64"

service:
  type: LoadBalancer
  port: 14128

metrics:
  cacheMetricsStorageSize: "1g"
  throughputHistogramBuckets: "50000000, 75000000, 100000000, 150000000, 200000000"

nucleus:
  enabled: true

monitoring:
  enabled: true
```

---

## Troubleshooting

### Common Issues and Solutions

#### 1. Pods Not Starting

**Symptoms:**
- Pods stuck in `Pending` or `ContainerCreating`
- PVCs not binding

**Diagnosis:**
```bash
# Check pod status
kubectl -n container-caching describe pod nvcf-container-cache-0

# Check PVC status
kubectl -n container-caching get pvc

# Check storage class
kubectl get storageclass
```

**Solutions:**
- Verify storage class exists and is available
- Check node resources and scheduling constraints
- Verify pull secrets are correct
- Check for insufficient node resources

#### 2. Cache Not Working

**Symptoms:**
- All requests show MISS in logs
- No performance improvement

**Diagnosis:**
```bash
# Check cache configuration
kubectl -n container-caching logs nvcf-container-cache-0 -c nginx-proxy | grep -i cache

# Check containerd configuration
kubectl -n container-caching exec ds/nvcf-container-cache-cc -- \
  cat /host/etc/containerd/config.toml
```

**Solutions:**
- Verify DaemonSet has configured containerd
- Check that cache service is accessible from nodes
- Restart containerd on worker nodes if needed
- Verify cache directory has sufficient space

#### 3. Authentication Issues

**Symptoms:**
- HTTP 401/403 errors
- Cannot pull private images

**Diagnosis:**
```bash
# Test authentication
./container-cache-test.sh

# Check NGC credentials
docker login nvcr.io
```

**Solutions:**
- Verify NGC API key is valid
- Check pull secrets are correctly created
- Ensure OAuth token exchange is working

#### 4. Performance Issues

**Symptoms:**
- High memory usage
- Slow response times
- Cache misses

**Diagnosis:**
```bash
# Check resource usage
kubectl -n container-caching top pods

# Check cache statistics
kubectl -n container-caching exec nvcf-container-cache-0 -c nginx-proxy -- \
  wget -qO- http://localhost:13128/stub_status
```

**Solutions:**
- Increase resource limits
- Adjust cache size and worker connections
- Consider disabling monitoring for resource optimization
- Check storage I/O performance

#### 5. Storage Issues

**Symptoms:**
- PVCs not binding
- Out of disk space
- Cache cleanup not working

**Diagnosis:**
```bash
# Check PVC status
kubectl -n container-caching get pvc

# Check disk usage
kubectl -n container-caching exec nvcf-container-cache-0 -- df -h

# Check storage class
kubectl describe storageclass <storage-class-name>
```

**Solutions:**
- Increase PVC size
- Adjust `freeProxyPct` for more aggressive cleanup
- Verify storage class supports dynamic provisioning
- Check node disk space if using EmptyDir

### Debug Commands

```bash
# View detailed pod information
kubectl -n container-caching describe pod nvcf-container-cache-0

# Check nginx configuration
kubectl -n container-caching exec nvcf-container-cache-0 -c nginx-proxy -- \
  nginx -T

# View nginx error logs
kubectl -n container-caching logs nvcf-container-cache-0 -c nginx-proxy | grep ERROR

# Check service endpoints
kubectl -n container-caching get endpoints nvcf-container-cache

# Test internal connectivity
kubectl -n container-caching exec nvcf-container-cache-0 -- \
  curl -k https://nvcf-container-cache:30345/v2/

# Monitor real-time cache activity
kubectl -n container-caching logs -f nvcf-container-cache-0 -c nginx-proxy | \
  grep -E '"upstream_cache_status":"(HIT|MISS)"'
```

---

## Advanced Configuration

### Custom nginx Configuration

To customize nginx behavior, modify the configuration files in `deploy/files/`:

- `nginx.conf`: Main nginx configuration
- `container-cache.conf`: Container registry caching logic
- `proxy-common.conf`: Common proxy settings
- `proxy-cache.conf`: S3 proxy cache configuration

### SSL/TLS Configuration

#### Custom Certificates (Without Vault)

```yaml
# Mount custom certificates
# Create ConfigMap with certificates
kubectl -n container-caching create configmap custom-certs \
  --from-file=tls.crt=/path/to/certificate.crt \
  --from-file=tls.key=/path/to/private.key

# Reference in values.yaml (requires template modification)
```

#### Certificate Rotation

When using Vault, certificates are automatically rotated. For custom certificates:

```bash
# Update certificate ConfigMap
kubectl -n container-caching create configmap custom-certs \
  --from-file=tls.crt=/path/to/new-certificate.crt \
  --from-file=tls.key=/path/to/new-private.key \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods to pick up new certificates
kubectl -n container-caching rollout restart statefulset nvcf-container-cache
```

### Performance Optimization

#### High-Throughput Configuration

```yaml
cache:
  workerConnection: 16384
  keyStorageSize: "1g"
  maxSize: "10000g"
  http2: "on"

resources:
  requests:
    memory: "64Gi"
    cpu: "16"
  limits:
    memory: "128Gi"
    cpu: "32"

# Use high-performance storage
persistentVolumeClaim:
  storageClassName: "premium-ssd"  # or equivalent high-IOPS class
  sizeGB: 5000
  sizeProxyGB: 10000

# Target high-performance nodes
nodeSelector:
  node.kubernetes.io/instance-type: "c5n.4xlarge"  # High network performance
```

#### Memory Optimization

```yaml
# Disable monitoring to save memory
monitoring:
  enabled: false

# Reduce cache size
cache:
  keyStorageSize: "10m"
  maxSize: "100g"
  workerConnection: 1000

# Lower resource requests
resources:
  requests:
    memory: "2Gi"
    cpu: "1"
  limits:
    memory: "8Gi"
    cpu: "4"
```

### Multi-Cluster Deployment

For deploying across multiple clusters with shared configuration:

```bash
# Use environment-specific values files
helm install nvcf-container-cache deploy/ \
  --namespace container-caching \
  --values deploy/values-gcp-ct1.yaml \
  --set vault.clusterName=gcp-cluster-1

helm install nvcf-container-cache deploy/ \
  --namespace container-caching \  
  --values deploy/values-aws-dev-1.yaml \
  --set vault.clusterName=aws-cluster-1
```

### Monitoring Integration

#### Prometheus Configuration

The cache exposes metrics on port 9113 when monitoring is enabled:

```yaml
# ServiceMonitor for Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nvcf-container-cache
  namespace: container-caching
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: nvcf-container-cache
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

#### Grafana Dashboard

Key metrics to monitor:
- `nginx_cache_requests_total`: Cache hit/miss ratio
- `nginx_connections_active`: Active connections
- `nginx_cache_size_bytes`: Cache storage usage
- Container pull times and success rates

---

## Uninstallation

### 1. Remove Helm Deployment

```bash
# Uninstall Helm release
helm uninstall nvcf-container-cache -n container-caching
```

### 2. Clean Up Node Configuration

The DaemonSet configures containerd on worker nodes. Clean up this configuration:

```bash
# Deploy cleanup DaemonSet
kubectl apply -f client/remove/remove-containerd-configuration.yaml

# Wait for completion
kubectl wait --for=condition=complete job/remove-containerd-config --timeout=300s

# Remove cleanup resources
kubectl delete -f client/remove/remove-containerd-configuration.yaml
```

### 3. Remove Persistent Data

```bash
# Delete PVCs (WARNING: This deletes all cached data)
kubectl -n container-caching delete pvc --all

# Delete namespace
kubectl delete namespace container-caching
```

### 4. Restart Containerd (If Needed)

On worker nodes, restart containerd to ensure clean state:

```bash
# On each worker node
sudo systemctl restart containerd
```

---

## Support and Resources

### Documentation Files

| File | Description |
|------|-------------|
| `deploy/values.yaml` | Default configuration values |
| `deploy/values-*.yaml` | Environment-specific examples |
| `deploy/Chart.yaml` | Helm chart metadata |
| `deploy/templates/` | Kubernetes resource templates |
| `tests/container-cache/` | Testing framework and scripts |

### Monitoring and Debugging

- **Prometheus Metrics**: Available at `http://pod-ip:9113/metrics`
- **Nginx Statistics**: Available at `http://pod-ip:13128/stub_status`  
- **Cache Logs**: Use `kubectl logs` with container name `nginx-proxy`

### Performance Tuning

- **Cache Hit Ratio**: Target >80% for production workloads
- **Memory Usage**: Monitor for cache efficiency vs resource usage
- **Storage I/O**: Use high-performance storage classes for better performance
- **Network**: Consider node placement for optimal network performance

### Contact Information

For issues, feature requests, or questions:
- **NVCF Infrastructure Team**: Internal support channel
- **Documentation**: This guide and inline comments in configuration files
- **Testing**: Use provided test framework for validation

---

**End of Document**

*This document covers the complete configuration, deployment, and testing of the NVCF Container Cache system. For the latest updates and additional configuration options, refer to the Helm chart templates and example values files.*
