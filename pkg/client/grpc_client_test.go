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
	"net"
	"testing"

	eventpb "github.com/incidentassistant/k8s-agent/proto/event"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

type server struct {
	eventpb.UnimplementedEventServiceServer
}

func (s *server) EmitEvent(ctx context.Context, message *eventpb.EventMessage) (*eventpb.EventResponse, error) {
	return &eventpb.EventResponse{Acknowledged: true}, nil
}

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	eventpb.RegisterEventServiceServer(s, &server{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic("Server exited with error")
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestSendEvent(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := eventpb.NewEventServiceClient(conn)

	resp, err := client.EmitEvent(ctx, &eventpb.EventMessage{})
	if err != nil {
		t.Fatalf("EmitEvent failed: %v", err)
	}
	if !resp.Acknowledged {
		t.Fatalf("EmitEvent response not acknowledged")
	}
}
