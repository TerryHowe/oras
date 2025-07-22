helm uninstall -n container-caching nvcf-container-cache
helm install nvcf-container-cache deploy/ -f deploy/values-stage.yaml -n container-caching
