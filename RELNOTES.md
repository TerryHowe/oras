# NVCF Proxy Cache Release Notes

# version 0.18.0
1. https://jirasw.nvidia.com/browse/NVCF-4700
2. https://jirasw.nvidia.com/browse/NVCF-4632

## Upgrade Instructions
The following configuration needs to be added to all Forge (AZ*) deployments

nucleus:
  enabled: true

# version 0.17.0
1. https://jirasw.nvidia.com/browse/NVCF-4025

## Upgrade Instructions
None

# version 0.16.0
1. https://nvbugspro.nvidia.com/bug/5190231
2. https://nvbugspro.nvidia.com/bug/5154038
3. https://jirasw.nvidia.com/browse/NVCF-2182

## Upgrade Instructions
None

# version 0.15.0
1. [OMPE-36465](https://jirasw.nvidia.com/browse/OMPE-36465) - Log the `ETag` of the object and whether the HTTP `Authorization` header was specified. Fix JSON syntax.
2. https://jirasw.nvidia.com/browse/NVCF-2604 - OTEL integration
3. https://nvbugspro.nvidia.com/bug/5108024 - [NVCF][Cluster Validation][az28] Cosmos Video Curator test pipeline failed due to fail to write data to s3
4. [OMPE-37543](https://jirasw.nvidia.com/browse/OMPE-37543) - No Access Checks when `Cache-Control: no-cache` is present.
5. [OMPE-28867]/[NVCF-2182] Adding Nucleus specific LFT proxy cache and service
6. [OMPE-36697] Adding logging to proxy-cache on the Omniverse side

## Upgrade Instructions:
```
Configuration Changes:
1. Add the following to values-*.yaml under images:
   collector: nvcr.io/nv-ngc-devops/opentelemetry-collector-contrib:0.112.0
   2. Add the following section to vaules-*.yaml
      traces:
       # Enable tracing
       enabled: false
        # OpenTelemetry-Collector configuration.
       collector:
        # OpenTelemetry endpoint to send traces to.
            endpoint: "prod.otel.kaizen.nvidia.com:8282"
    
```

# Version 0.14.0
1. https://nvbugspro.nvidia.com/bug/5062467 - Retry all errors atleast once.

## Upgrade Instructions:
None

# Version 0.13.0
1. https://nvbugspro.nvidia.com/bug/5062467 - Storage failures due to proxy cache causes function errors

## Upgrade Instructions:
None

# Version 0.12.0
1. https://jirasw.nvidia.com/browse/NVCFSRE-3029 - Publish a `proxy_cache_instance_info` gauge metric to expose chart version for dashboarding

## Upgrade Instructions:
None

# Version 0.11.0
1. https://nvbugspro.nvidia.com/bug/5054765 - Set client_max_body_size to 0. Set proxy_request_buffering off.

# Upgrade Instructions:
None

# Version 0.10.0
1. Use `min_free` instead of `max_size` in nginx.conf's `proxy_cache` directive to ensure safe utilization of available disk space
and adjust value files accordingly for all clusters. This is part of alert review, https://jirasw.nvidia.com/browse/NVCFSRE-1928.

# Upgrade Instructions:
None

# Version 0.9.0
1. https://nvbugspro.nvidia.com/bug/5025641 - nvcf-container-cache container update to v1.1.23. 
Adjusted slice_range and http_range processing.

# Upgrade Instructions:
None

# Version 0.8.0
1. https://jirasw.nvidia.com/browse/OMPE-31483 - nvcf-container-cache container update to v1.1.22
2. https://nvbugspro.nvidia.com/bug/5023141 - cache refresh when data in S3 is different from data in cache.

# Upgrade Instructions:
None

# Version 0.7.0
1. https://nvbugspro.nvidia.com/bug/5008002 - Adjust certificate validity time to account for clock skew between pods. 

# Upgrade Instructions:
None
