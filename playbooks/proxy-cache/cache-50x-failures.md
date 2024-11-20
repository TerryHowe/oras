# 50x cache failures

Follow this runbook to remediate and investigate issues leading to cache failures with HTTP 50x error codes.

## Disable the cache on the cluster

The best way to disable the cache right now is to change on the fly the unbound configuration and prevent further redirects to the cache.

This **does not work** in `az27` due to the enforcement of argocd sync, so if we need to change something there, escalate to DGXC so they can disable it.

The change can be done quickly by running:

```
kubectl get cm -n dns-proxy nvcf-unbound-config \
  | sed -r '/.*local-(zone|data):.*s3.*/d' \
  | kubectl apply -f -
```

This will disable the configuration for all `s3` buckets using the cache, without requiring functions are restarted. 
After you're sure the configmap looks ok, you can run:

```
kubectl rollout restart statefulset -n dns-proxy nvcf-unbound
```

Which will restart the UnboundDNS pods. The cache should now be disabled and you will get time for further debugging of 
the issue. Do note some function may have unacceptable performance with the cache disabled (due to slow loading of assets 
on repeated loads) so this is *not* a panacea.

If you can use logs to determine quickly the requests impacted relate only to a single zone (ex: for a specific S3 bucket)
 is impacted, you may as well just remove the `local-zone` and `local-data` lines for that zone from the server configmap 
 (`dns-proxy`/`nvcf-unbound-config`) and execute a `rollout restart` of the unbound statefulset as above instead of 
 deleting them all.
