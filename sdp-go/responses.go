package sdp

// TODO: instead of translating, unify this
func (r *Response) ToQueryStatus() *QueryStatus {
	return &QueryStatus{
		UUID:   r.GetUUID(),
		Status: r.GetState().ToQueryStatus(),
	}
}

// TODO: instead of translating, unify this
func (r ResponderState) ToQueryStatus() QueryStatus_Status {
	switch r {
	case ResponderState_WORKING:
		return QueryStatus_STARTED
	case ResponderState_COMPLETE:
		return QueryStatus_FINISHED
	case ResponderState_ERROR:
		return QueryStatus_ERRORED
	case ResponderState_CANCELLED:
		return QueryStatus_CANCELLED
	case ResponderState_STALLED:
		return QueryStatus_ERRORED
	default:
		return QueryStatus_UNSPECIFIED
	}
}
