# Copyright 2024 Incident Assistant AI
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Start from the Golang base image
FROM golang:1.22-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Install Git (required for go mod download)
RUN apk add --no-cache git

# Copy go.mod and go.sum to download the dependencies
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application with flags to reduce the binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o controller ./cmd/controller

# Start a new stage from scratch for a smaller final image
FROM scratch

# Add ca-certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set the working directory to /root/
WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/controller .

# Command to run the executable
CMD ["./controller"]
