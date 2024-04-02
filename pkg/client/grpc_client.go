// Copyright 2024 Incident Assistant AI
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure" // Import the insecure package

	eventpb "github.com/incidentassistant/k8s-agent/proto/event"
)

// NewEventServiceClient creates a new client for the EventService
func NewEventServiceClient() (eventpb.EventServiceClient, error) {
	destinationURL := os.Getenv("DESTINATION_URL") // Central hub URL
	useTLS := os.Getenv("USE_TLS") == "true"       // New environment variable to control TLS usage

	var opts []grpc.DialOption
	if useTLS {
		// Use the system's default CA certificates
		creds := credentials.NewClientTLSFromCert(nil, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		// Use insecure credentials
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials())) // Updated line
	}

	// Set up a connection to the server
	conn, err := grpc.Dial(destinationURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not connect to gRPC server: %w", err)
	}

	// Create a new EventService client
	client := eventpb.NewEventServiceClient(conn)
	return client, nil
}

// SendEvent sends an event to the EventService
func SendEvent(client eventpb.EventServiceClient, eventMessage *eventpb.EventMessage) (*eventpb.EventResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.EmitEvent(ctx, eventMessage)
	if err != nil {
		return nil, fmt.Errorf("could not send event: %w", err)
	}
	return response, nil
}
