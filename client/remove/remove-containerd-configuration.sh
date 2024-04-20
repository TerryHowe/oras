# Copyright (c) 2023-2024, NVIDIA CORPORATION.  All rights reserved.
#
# NVIDIA CORPORATION and its licensors retain all intellectual property
# and proprietary rights in and to this software, related documentation
# and any modifications thereto.  Any use, reproduction, disclosure or
# distribution of this software and related documentation without an express
# license agreement from NVIDIA CORPORATION is strictly prohibited.

kubectl create -f ./client/remove/remove-containerd-configuration.yaml
kubectl wait --for=condition=ready pod -l name=remove-containerd-configuration -n kube-system --timeout=60s
kubectl delete -f ./client/remove/remove-containerd-configuration.yaml