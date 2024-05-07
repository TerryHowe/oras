# Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
#
# NVIDIA CORPORATION and its licensors retain all intellectual property
# and proprietary rights in and to this software, related documentation
# and any modifications thereto.  Any use, reproduction, disclosure or
# distribution of this software and related documentation without an express
# license agreement from NVIDIA CORPORATION is strictly prohibited.

ARG TAG=v3.0.6-dev
ARG IMG=nvcr.io/nvidian/distroless/go
FROM golang:1.19 as builder

ENV DEBIAN_FRONTEND=noninteractive

ENV CGO_ENABLED=0

WORKDIR /app

COPY . .

# build our go executable
RUN go build

FROM $IMG:$TAG

# Copy the go executable into our final container
COPY --from=builder /app/cert-watcher /app/cert-watcher

CMD ["/app/cert-watcher"]
