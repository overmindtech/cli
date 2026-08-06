package sdp

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestPreferredIDEContractIsRemovedAndReserved(t *testing.T) {
	t.Parallel()

	service := File_brent_proto.Services().ByName("WorkspaceService")
	if service == nil {
		t.Fatal("WorkspaceService descriptor missing")
	}
	if service.Methods().ByName("SetPreferredIDE") != nil {
		t.Fatal("WorkspaceService must not expose SetPreferredIDE")
	}

	messages := File_brent_proto.Messages()
	if messages.ByName("SetPreferredIDERequest") != nil {
		t.Fatal("SetPreferredIDERequest must not exist")
	}
	if messages.ByName("SetPreferredIDEResponse") != nil {
		t.Fatal("SetPreferredIDEResponse must not exist")
	}

	principalStatus := messages.ByName("GetPrincipalStatusResponse")
	if principalStatus.Fields().ByNumber(4) != nil {
		t.Fatal("GetPrincipalStatusResponse field 4 must not exist")
	}
	if principalStatus.Fields().ByName("preferred_ide") != nil {
		t.Fatal("GetPrincipalStatusResponse preferred_ide must not exist")
	}
	if !principalStatus.ReservedRanges().Has(protoreflect.FieldNumber(4)) {
		t.Fatal("GetPrincipalStatusResponse field 4 must be reserved")
	}
	if !principalStatus.ReservedNames().Has("preferred_ide") {
		t.Fatal("GetPrincipalStatusResponse preferred_ide name must be reserved")
	}
}

// TestENG5923FieldNumbersUnchanged pins the representative field / oneof /
// option numbers the Until rename must preserve (ENG-5960 Part 2).
func TestENG5923FieldNumbersUnchanged(t *testing.T) {
	t.Parallel()

	messages := File_brent_proto.Messages()

	t.Run("blocking_difference_tags_1_2_3", func(t *testing.T) {
		t.Parallel()
		var msg protoreflect.MessageDescriptor
		for i := range messages.Len() {
			m := messages.Get(i)
			if m.Fields().ByName("blocking_difference_tags") != nil &&
				m.Fields().ByName("blocking_difference_tags_updated_by") != nil {
				msg = m
				break
			}
		}
		if msg == nil {
			t.Fatal("no message carries blocking_difference_tags fields 1-3")
		}
		assertFieldNumber(t, msg, "blocking_difference_tags", 1)
		assertFieldNumber(t, msg, "blocking_difference_tags_updated_by", 2)
		assertFieldNumber(t, msg, "blocking_difference_tags_updated_at", 3)
	})

	t.Run("event_oneof_plan_check_completed_28_loop_run_queued_30", func(t *testing.T) {
		t.Parallel()
		event := messages.ByName("Event")
		if event == nil {
			t.Fatal("Event message missing")
		}
		assertFieldNumber(t, event, "plan_check_completed", 28)
		assertFieldNumber(t, event, "loop_run_queued", 30)
	})

	t.Run("applied_blocking_difference_tags_10", func(t *testing.T) {
		t.Parallel()
		msg := messages.ByName("PlanCheckCompleted")
		if msg == nil {
			t.Fatal("PlanCheckCompleted message missing")
		}
		assertFieldNumber(t, msg, "applied_blocking_difference_tags", 10)
	})

	t.Run("embedded_json_option_50001", func(t *testing.T) {
		t.Parallel()
		github := messages.ByName("GitHubWebhook")
		if github == nil {
			t.Fatal("GitHubWebhook message missing")
		}
		field := github.Fields().ByName("event_payload")
		if field == nil {
			t.Fatal("GitHubWebhook.event_payload missing")
		}
		optsMsg := field.Options().(*descriptorpb.FieldOptions)
		if optsMsg == nil {
			t.Fatal("event_payload has no field options")
		}
		if E_EmbeddedJson.TypeDescriptor().Number() != 50001 {
			t.Fatalf("E_EmbeddedJson number=%d want 50001", E_EmbeddedJson.TypeDescriptor().Number())
		}
		val := proto.GetExtension(optsMsg, E_EmbeddedJson)
		got, ok := val.(bool)
		if !ok || !got {
			t.Fatalf("embedded_json extension: got %v (%T), want true", val, val)
		}
	})
}

func assertFieldNumber(t *testing.T, msg protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber) {
	t.Helper()
	field := msg.Fields().ByName(name)
	if field == nil {
		t.Fatalf("%s.%s missing", msg.Name(), name)
	}
	if field.Number() != number {
		t.Fatalf("%s.%s number=%d want %d", msg.Name(), name, field.Number(), number)
	}
}
