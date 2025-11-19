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

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Run stage
FROM golang:1.25-alpine

# Install runtime dependencies including build tools for CGO (required for -race flag)
RUN apk add --no-cache ca-certificates git curl gcc musl-dev

# Set working directory
WORKDIR /workspace

# Copy from builder
COPY --from=builder /go/pkg /go/pkg
COPY --from=builder /workspace /workspace

# Default command runs e2e tests
CMD ["go", "test", "-mod=mod", "-v", "-race", "./test/e2e/..."]
