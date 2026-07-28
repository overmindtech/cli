package sdp

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPreferredIDEContractIsRemovedAndReserved(t *testing.T) {
	t.Parallel()

	service := File_brent_proto.Services().ByName("BrentService")
	if service.Methods().ByName("SetPreferredIDE") != nil {
		t.Fatal("BrentService must not expose SetPreferredIDE")
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
