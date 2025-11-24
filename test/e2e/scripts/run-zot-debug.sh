#!/bin/bash
# Copyright The ORAS Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

echo "Starting zot-debug pod interactively in Kubernetes..."
echo ""

# Check if namespace exists
if ! kubectl get namespace oras-e2e-tests &> /dev/null; then
    echo "Error: Namespace 'oras-e2e-tests' does not exist."
    echo "Please deploy the registries first:"
    echo "  ./test/e2e/scripts/deploy.sh"
    exit 1
fi

# Check if zot-registry-pvc exists
if ! kubectl get pvc zot-registry-pvc -n oras-e2e-tests &> /dev/null; then
    echo "Error: PVC 'zot-registry-pvc' does not exist."
    echo "Please ensure zot registry is deployed:"
    echo "  ./test/e2e/scripts/deploy.sh"
    exit 1
fi

# Clean up any existing debug pod
POD_NAME="zot-debug"
if kubectl get pod "$POD_NAME" -n oras-e2e-tests &> /dev/null; then
    echo "Cleaning up existing debug pod..."
    kubectl delete pod "$POD_NAME" -n oras-e2e-tests --wait=true
fi

echo "Creating zot-debug pod..."
echo ""

# Apply the debug pod configuration
kubectl apply -f test/e2e/k8s/zot-debug-pod.yaml

# Wait for pod to be ready
echo "Waiting for pod to be ready..."
kubectl wait --for=condition=Ready --timeout=60s pod/"$POD_NAME" -n oras-e2e-tests

echo ""
echo "=================================================="
echo "zot-debug pod is ready!"
echo ""
echo "Volume mount:"
echo "  /etc/zot (read-only) -> zot-registry-pvc"
echo ""
echo "Useful commands:"
echo "  ls -la /etc/zot"
echo "  find /etc/zot -type f"
echo "  du -sh /etc/zot"
echo ""
echo "To exit the shell, type 'exit' or press Ctrl+D"
echo ""
echo "The pod will be automatically deleted when you exit."
echo "=================================================="
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up debug pod..."
    kubectl delete pod "$POD_NAME" -n oras-e2e-tests --wait=false &> /dev/null || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Connect to the pod interactively
kubectl exec -it "$POD_NAME" -n oras-e2e-tests -- /bin/sh
