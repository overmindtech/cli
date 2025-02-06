package sdp

import "testing"

func TestEqual(t *testing.T) {
	x := &GatewayRequestStatus{
		ResponderStates: map[string]ResponderState{
			"foo": ResponderState_COMPLETE,
			"boo": ResponderState_WORKING,
			"bar": ResponderState_ERROR,
		},
		Summary: &GatewayRequestStatus_Summary{
			Working:    1,
			Stalled:    0,
			Complete:   1,
			Error:      1,
			Cancelled:  0,
			Responders: 3,
		},
	}

	t.Run("with nil summary", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with nil ResponderStates", func(t *testing.T) {
		y := &GatewayRequestStatus{
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with mismatched summary", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   3,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with mismatched ResponderStates", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"BOO": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with different postprocessing states", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
			PostProcessingComplete: true,
		}

		if x.Equal(y) {
			t.Error("expected items to be different")
		}
	})

	t.Run("with same everything", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if !x.Equal(y) {
			t.Error("expected items to be equal")
		}
	})
}

func TestDone(t *testing.T) {
	t.Run("with a request that should be done", func(t *testing.T) {
		r := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_STALLED,
				"bar": ResponderState_ERROR,
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    0,
				Stalled:    1,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
			PostProcessingComplete: true,
		}

		if !r.Done() {
			t.Error("expected request .Done() to be true")
		}
	})

	t.Run("with a request that shouldn't be done", func(t *testing.T) {
		r := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
			PostProcessingComplete: false,
		}

		if r.Done() {
			t.Error("expected request .Done() to be false")
		}

		r.PostProcessingComplete = true

		if r.Done() {
			t.Error("expected request .Done() to be false")
		}
	})
}
