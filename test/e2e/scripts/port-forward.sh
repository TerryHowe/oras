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

# Check if namespace exists
if ! kubectl get namespace oras-e2e-tests &> /dev/null; then
    echo "Error: Namespace 'oras-e2e-tests' does not exist."
    echo "Run './test/e2e/scripts/deploy.sh' to deploy the registries first."
    exit 1
fi

echo "Setting up port forwarding for ORAS e2e test registries..."
echo ""
echo "Local endpoints will be:"
echo "  Docker Registry:   localhost:5000"
echo "  Fallback Registry: localhost:6000"
echo "  Zot Registry:      localhost:7000"
echo ""
echo "Press Ctrl+C to stop port forwarding"
echo ""

# Trap Ctrl+C to cleanup background processes
trap 'echo ""; echo "Stopping port forwarding..."; kill $(jobs -p) 2>/dev/null; exit' INT TERM

# Port forward Docker Registry
kubectl port-forward -n oras-e2e-tests service/docker-registry 5000:5000 &
DOCKER_PF_PID=$!

# Port forward Fallback Registry
kubectl port-forward -n oras-e2e-tests service/fallback-registry 6000:5000 &
FALLBACK_PF_PID=$!

# Port forward Zot Registry
kubectl port-forward -n oras-e2e-tests service/zot-registry 7000:5000 &
ZOT_PF_PID=$!

# Wait for port forwards to be established
sleep 2

# Check if port forwards are working
if ! curl -s http://localhost:5000/v2/ > /dev/null 2>&1; then
    echo "Warning: Docker Registry port forward may not be ready yet"
fi

if ! curl -s http://localhost:6000/v2/ > /dev/null 2>&1; then
    echo "Warning: Fallback Registry port forward may not be ready yet"
fi

if ! curl -s http://localhost:7000/v2/ > /dev/null 2>&1; then
    echo "Warning: Zot Registry port forward may not be ready yet"
fi

echo "Port forwarding is active!"
echo ""

# Wait indefinitely
wait
