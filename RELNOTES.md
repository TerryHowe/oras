# NVCF Proxy Cache Release Notes

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
