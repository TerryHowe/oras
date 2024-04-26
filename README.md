# Container Caching


### Overview

This project provides a container caching solution specifically optimized for NGC. It is designed to enhance the efficiency of Docker image pulls from NGC by caching the images locally, reducing network bandwidth and improving pull times for frequently accessed images.

### Pre-requisite

1. The container image for Container Cache is stored on NGC at [nvcr.io/nvstaging/clara/nvcf-container-caching](https://registry.ngc.nvidia.com/orgs/nvstaging/teams/clara/containers/nvcf-container-caching). Make sure you have access to it.

2. Kubernetes Cluster with containerd runtime. Container Cache is build for kubernetes deployment. Client nodes that need to pull images from the Cache must be using containerd runtime.

### Setup Instructions

See [deploy](./deploy/) for container cache helm chart

1. Create `container-caching` namespace
    
    Vault is configured to select the select the ServiceAccount in `container-caching` namespace. The helm chart needs to be deployed to this namespace for vault to allow fetching certificates for nginx. Use below command to create namespace:

    ```
    kubectl create namespace container-caching
    ```

2. Create image pull secret 

    Container Cache Image is present on [NGC](https://registry.ngc.nvidia.com/orgs/qtfpt1h0bieu/teams/nvcf-core/containers/nvcf-container-cache/tags). 
    Image pull secret with your NGC key must be created in the `container-caching` namespace. Below is the sample command to create secret:
    ```
    kubectl create secret docker-registry ngc-container-pull --docker-server=nvcr.io --docker-username='$oauthtoken' --docker-password=<your-ngc-key> -n container-caching
    ```

3. Customizing the helm [values.yaml](./deploy/)

    Modify the values present in [values.yaml](./deploy/values-azure-stage.yaml) as per requirement.

4. Deploy Helm Chart

    Use the below command to deploy helm chart
    ```
    helm install nvcf-container-cache-staging deploy/ -f deploy/values.yaml -n container-caching
    ```

5. Verify deployment

    Use the below command to verify the deployment 
    ```
    kubectl --namespace=container-caching get pod,daemonset,svc,deployment,secret,pvc --selector='app.kubernetes.io/name=nvcf-container-cache,app.kubernetes.io/instance=<helm-release-name>'

    NAME                                     READY   STATUS    RESTARTS   AGE
    pod/nvcf-container-cache-nvmesh-nvcr-0   3/3     Running   0          10s

    NAME                                                 DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
    daemonset.apps/nvcf-container-cache-nvmesh-nvcr-cc   12        12        12      12           12          <none>          11s

    NAME                                       TYPE       CLUSTER-IP       EXTERNAL-IP   PORT(S)           AGE
    service/nvcf-container-cache-nvmesh-nvcr   NodePort   100.68.193.167   <none>        30345:30345/TCP   11s

    NAME                                                READY   AGE
    statefulset.apps/nvcf-container-cache-nvmesh-nvcr   1/1     12s

    NAME                                                             STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
    persistentvolumeclaim/cache-nvcf-container-cache-nvmesh-nvcr-0   Bound    pvc-e4c36637-dee5-49b5-8267-9b37cf38409a   20Gi       RWO            nvcf-cc-sc      39h   
    ```

6. The Helm Chart includes a [daemonset](./deploy/templates/daemonset.yaml) that does containerd configuration on each node. The pods spawned by daemonset remain in `Running` state. As nodes are created, new pod gets spawned to make containerd configuration.

### Uninstall Instructions

1. Uninstall Helm Chart

    ```
    helm uninstall nvcf-container-cache-nvmesh-nvcr -n container-caching
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
