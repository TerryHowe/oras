# Functions report SSL CERTIFICATE_VERIFY_FAILED

Applications may fail with something similar to:

```
requests.exceptions.SSLError: HTTPSConnectionPool(host='proxy-cache-validate.s3.us-east-1.amazonaws.com', port=443): Max retries exceeded with url: /nv-gb200.jpg (Caused by SSLError(SSLCertVerificationError(1, '[SSL: CERTIFICATE_VERIFY_FAILED] certificate verify failed: self signed certificate in certificate chain (_ssl.c:1000)')))
```

The above is from a python application, but application in other languages will have similar errors.

We provide the needed CA certificates under `/etc/ssl/certs/` (see [the `add-certificates-volume` kyverno rule](https://gitlab-master.nvidia.com/nvcf/nvcf-cache/kyverno/-/blob/main/kyverno-cert-mount.yaml)).

## Verify there are no errors in applying the Kyverno ClusterPolicy

You can use the following query to check in the [NVCF Thanos Grafana](https://nvcf-grafana.thanos.nvidiangn.net/) (using the `Explore` section if there were any issues in a specific cluster):

```
sum(increase(kyverno_policy_results_total{
        rule_execution_cause="admission_request", 
        policy_name="add-certificates-volume", 
        cluster=~"[cluster]", 
        rule_result!~"pass|skip"
}[5m])) by (rule_result, rule_name)
```

This should return no data. If errors are returned, you will need to dig into `kyverno` pod logs to figure out why some rules are not getting applied.

## Verify application is configured properly

Check the application is properly using and importing our CA certificates. 

Python applications will use an embedded CA bundle (normally from the `python-certifi` package). Those applications should have the following variable set up as part of the function that run them (or any other way of initialization):

```
export REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt
```