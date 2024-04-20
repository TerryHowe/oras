# Container Caching


### Overview

This project provides a container caching solution specifically optimized for NGC. It is designed to enhance the efficiency of Docker image pulls from NGC by caching the images locally, reducing network bandwidth and improving pull times for frequently accessed images.

### Pre-requisite

1. The container image for Container Cache is stored on NGC at [nvcr.io/nvstaging/clara/nvcf-container-caching](https://registry.ngc.nvidia.com/orgs/nvstaging/teams/clara/containers/nvcf-container-caching). Make sure you have access to it.

2. Kubernetes Cluster with containerd runtime. Container Cache is build for kubernetes deployment. Client nodes that need to pull images from the Cache must be using containerd runtime.

### Setup Instructions

#### Server
See [deploy](./deploy/) for container cache helm chart

1. Create `container-caching` namespace
    
    Vault is configured to select the select the ServiceAccount in `container-caching` namespace. The helm chart needs to be deployed to this namespace for vault to allow fetching certificates for nginx. Use below command to create namespace:

    ```
    kubectl create namespace container-caching
    ```

2. Create image pull secret 

    Container Cache Image is present on [NGC](https://registry.ngc.nvidia.com/orgs/nvstaging/teams/clara/containers/nvcf-container-caching). 
    Image pull secret with your NGC key must be created in the `container-caching` namespace. Below is the sample command to create secret:
    ```
    kubectl create secret docker-registry ngc-container-pull --docker-server=nvcr.io --docker-username='$oauthtoken' --docker-password=<your-ngc-key> -n container-caching
    ```

3. Customizing the helm [values.yaml](./deploy/)

    Modify the values present in [values.yaml](./deploy/values.yaml) as per requirement.

4. Deploy Helm Chart

    Use the below command to deploy helm chart
    ```
    helm install nvcf-container-cache-staging deploy/ -f deploy/values.yaml -n container-caching
    ```

5. Verify deployment

    Use the below command to verify the deployment 
    ```
    kubectl --namespace=container-caching get pod,svc,deployment,secret,pvc --selector='app.kubernetes.io/name=nvcf-container-cache,app.kubernetes.io/instance=<helm-release-name>'   
    ```

#### Client

##### Configure Containerd using Daemonset
See [client](./client//) for container cache daemonset

1. Customize [daemonset](./client/configure-containerd.yaml)

    The daemonset configures Container Cache deployment as an image mirror for ${TARGET_HOST} registry for all the nodes in the cluster. 
    
    Modify ${TARGET_HOST} to your upstream server and ${CONTAINER_CACHE_IP} to the IP of your Container Cache deployment.

    Modify `nginx_proxy.crt` to the Root CA Cert with which was used to sign the Nginx SSL Cert.

2. Run [configure-containerd.sh](./client/configure-containerd.sh)
    
    The above script deploys the daemonset that adds containerd configuration for Container Cache, waits for it's completion and exits. Use below command to run the script:

    ```
    ./client/configure-containerd.sh
    ```

##### Remove Containerd Configuration using Daemonset
See [client](./client/remove/) for container cache configuration removal daemonset

1. Customize [daemonset](./client/remove/remove-containerd-configuration.sh)

    The daemonset removes Containerd configuration done for Container Cache for all the nodes in the cluster. 
    
    Modify ${TARGET_HOST} to your upstream server.

2. Run [remove-containerd-configuration.sh](./client/remove/remove-containerd-configuration.sh)
    
    The above script deploys the daemonset that removes containerd configuration, waits for it's completion and exits. Use below command to run the script:

    ```
    ./client/remove/remove-containerd-configuration.sh
    ```

### Additional Information

For more detailed information about the configuration and customization options, please refer to the provided configuration files and scripts in this repository.