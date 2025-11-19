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

echo "ORAS e2e Test Registries Status"
echo "================================"
echo ""

# Check if namespace exists
if ! kubectl get namespace oras-e2e-tests &> /dev/null; then
    echo "Namespace 'oras-e2e-tests' does not exist."
    echo "Run './test/e2e/scripts/deploy.sh' to deploy the registries."
    exit 0
fi

echo "Deployments:"
echo "------------"
kubectl get deployments -n oras-e2e-tests

echo ""
echo "Pods:"
echo "-----"
kubectl get pods -n oras-e2e-tests

echo ""
echo "Services:"
echo "---------"
kubectl get services -n oras-e2e-tests

echo ""
echo "PersistentVolumeClaims:"
echo "-----------------------"
kubectl get pvc -n oras-e2e-tests

echo ""
echo "Registry Health Checks:"
echo "-----------------------"

# Check Docker Registry
DOCKER_POD=$(kubectl get pods -n oras-e2e-tests -l app=docker-registry -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$DOCKER_POD" ]; then
    echo -n "Docker Registry v2: "
    if kubectl exec -n oras-e2e-tests "$DOCKER_POD" -- wget -q -O- http://localhost:5000/v2/ &> /dev/null; then
        echo "✓ Healthy"
    else
        echo "✗ Unhealthy"
    fi
fi

# Check Zot Registry
ZOT_POD=$(kubectl get pods -n oras-e2e-tests -l app=zot-registry -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$ZOT_POD" ]; then
    echo -n "Zot Registry:       "
    if kubectl exec -n oras-e2e-tests "$ZOT_POD" -- wget -q -O- http://localhost:5000/v2/ &> /dev/null; then
        echo "✓ Healthy"
    else
        echo "✗ Unhealthy"
    fi
fi
