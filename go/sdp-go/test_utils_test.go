package sdp

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

func TestRequest(t *testing.T) {
	tc := TestConnection{}

	t.Run("with a regular subject", func(t *testing.T) {
		// Create the responder
		_, err := tc.Subscribe("test", func(msg *nats.Msg) {
			err2 := tc.Publish(context.Background(), msg.Reply, &GatewayResponse{
				ResponseType: &GatewayResponse_Error{
					Error: "testing",
				},
			})
			if err2 != nil {
				t.Error(err2)
			}
		})
		if err != nil {
			t.Fatal(err)
		}

		request := &GatewayRequest{}

		data, err := proto.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		msg := nats.Msg{
			Subject: "test",
			Data:    data,
		}
		replyMsg, err := tc.RequestMsg(context.Background(), &msg)
		if err != nil {
			t.Fatal(err)
		}

		response := &GatewayResponse{}
		err = proto.Unmarshal(replyMsg.Data, response)
		if err != nil {
			t.Error(err)
		}

		if response.GetResponseType().(*GatewayResponse_Error).Error != "testing" {
			t.Errorf("expected error to be 'testing', got '%v'", response)
		}
	})

	t.Run("with a > wildcard subject", func(t *testing.T) {
		// Create the responder
		_, err := tc.Subscribe("test.>", func(msg *nats.Msg) {
			err2 := tc.Publish(context.Background(), msg.Reply, &GatewayResponse{
				ResponseType: &GatewayResponse_Error{
					Error: "testing",
				},
			})
			if err2 != nil {
				t.Error(err2)
			}
		})
		if err != nil {
			t.Fatal(err)
		}

		request := &GatewayRequest{}

		data, err := proto.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		msg := nats.Msg{
			Subject: "test.foo.bar",
			Data:    data,
		}
		replyMsg, err := tc.RequestMsg(context.Background(), &msg)
		if err != nil {
			t.Fatal(err)
		}

		response := &GatewayResponse{}
		err = proto.Unmarshal(replyMsg.Data, response)
		if err != nil {
			t.Error(err)
		}

		if response.GetResponseType().(*GatewayResponse_Error).Error != "testing" {
			t.Errorf("expected error to be 'testing', got '%v'", response)
		}
	})

	t.Run("with a * wildcard subject", func(t *testing.T) {
		// Create the responder
		_, err := tc.Subscribe("test.*.bar", func(msg *nats.Msg) {
			err2 := tc.Publish(context.Background(), msg.Reply, &GatewayResponse{
				ResponseType: &GatewayResponse_Error{
					Error: "testing",
				},
			})
			if err2 != nil {
				t.Error(err2)
			}
		})
		if err != nil {
			t.Fatal(err)
		}

		request := &GatewayRequest{}

		data, err := proto.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		msg := nats.Msg{
			Subject: "test.foo.bar",
			Data:    data,
		}
		replyMsg, err := tc.RequestMsg(context.Background(), &msg)
		if err != nil {
			t.Fatal(err)
		}

		response := &GatewayResponse{}
		err = proto.Unmarshal(replyMsg.Data, response)
		if err != nil {
			t.Error(err)
		}

		if response.GetResponseType().(*GatewayResponse_Error).Error != "testing" {
			t.Errorf("expected error to be 'testing', got '%v'", response)
		}
	})

}
