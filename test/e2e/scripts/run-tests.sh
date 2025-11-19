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

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="${SCRIPT_DIR}/../k8s"

# Check if namespace exists
if ! kubectl get namespace oras-e2e-tests &> /dev/null; then
    echo "Error: Namespace 'oras-e2e-tests' does not exist."
    echo "Please deploy the registries first:"
    echo "  ./test/e2e/scripts/deploy.sh"
    exit 1
fi

# Check if registries are ready
echo "Checking if registries are ready..."
if ! kubectl wait --for=condition=available --timeout=60s \
    deployment/docker-registry \
    deployment/fallback-registry \
    deployment/zot-registry \
    -n oras-e2e-tests &> /dev/null; then
    echo "Error: Registries are not ready."
    echo "Please ensure registries are deployed and running:"
    echo "  ./test/e2e/scripts/status.sh"
    exit 1
fi
echo "✓ Registries are ready"
echo ""

# Delete previous job if it exists
if kubectl get job oras-e2e-tests -n oras-e2e-tests &> /dev/null; then
    echo "Deleting previous test job..."
    kubectl delete job oras-e2e-tests -n oras-e2e-tests --wait=true
    echo "✓ Previous job deleted"
    echo ""
fi

# Create the job
echo "Creating e2e test job..."
kubectl apply -f "${K8S_DIR}/e2e-test-job.yaml"
echo "✓ Job created"
echo ""

# Wait for the pod to be created
echo "Waiting for test pod to be created..."
for i in $(seq 1 30); do
    POD_NAME=$(kubectl get pods -n oras-e2e-tests -l app=oras-e2e-tests \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [ -n "$POD_NAME" ]; then
        echo "✓ Test pod created: $POD_NAME"
        break
    fi
    sleep 1
done

if [ -z "$POD_NAME" ]; then
    echo "Error: Test pod was not created within timeout"
    exit 1
fi

echo ""
echo "==================================="
echo "Streaming test logs..."
echo "==================================="
echo ""

# Stream logs from the pod
# Wait for the pod to start and then follow logs
kubectl wait --for=condition=Ready --timeout=60s pod/"$POD_NAME" -n oras-e2e-tests 2>/dev/null || true
kubectl logs -f "$POD_NAME" -n oras-e2e-tests

echo ""
echo "==================================="
echo "Checking job status..."
echo "==================================="

# Wait for job to complete and get status
if kubectl wait --for=condition=complete --timeout=5m job/oras-e2e-tests -n oras-e2e-tests 2>/dev/null; then
    echo "✓ Tests passed successfully!"
    echo ""
    echo "To view logs again, run:"
    echo "  kubectl logs -n oras-e2e-tests job/oras-e2e-tests"
    exit 0
else
    # Check if job failed
    if kubectl wait --for=condition=failed --timeout=1s job/oras-e2e-tests -n oras-e2e-tests 2>/dev/null; then
        echo "✗ Tests failed"
        echo ""
        echo "To view logs again, run:"
        echo "  kubectl logs -n oras-e2e-tests job/oras-e2e-tests"
        echo ""
        echo "To debug the pod, run:"
        echo "  kubectl describe pod $POD_NAME -n oras-e2e-tests"
        exit 1
    else
        echo "✗ Job did not complete within timeout"
        echo ""
        echo "To check job status:"
        echo "  kubectl get job oras-e2e-tests -n oras-e2e-tests"
        echo ""
        echo "To view logs:"
        echo "  kubectl logs -n oras-e2e-tests job/oras-e2e-tests"
        exit 1
    fi
fi
