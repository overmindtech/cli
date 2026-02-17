package sdp

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetLogRecordsRequest_Validate(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)

	tests := []struct {
		name    string
		req     *GetLogRecordsRequest
		wantErr bool
	}{
		{
			name:    "Nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "Empty scope",
			req: &GetLogRecordsRequest{
				Scope:           "",
				Query:           "valid-query",
				From:            timestamppb.New(pastTime),
				To:              timestamppb.New(now),
				MaxRecords:      100,
				StartFromOldest: false,
			},
			wantErr: true,
		},
		{
			name: "Empty query",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "",
				From:            timestamppb.New(pastTime),
				To:              timestamppb.New(now),
				MaxRecords:      100,
				StartFromOldest: false,
			},
			wantErr: true,
		},
		{
			name: "Missing from timestamp",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            nil,
				To:              timestamppb.New(now),
				MaxRecords:      100,
				StartFromOldest: false,
			},
			wantErr: true,
		},
		{
			name: "Missing to timestamp",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            timestamppb.New(pastTime),
				To:              nil,
				MaxRecords:      100,
				StartFromOldest: false,
			},
			wantErr: true,
		},
		{
			name: "From after to",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            timestamppb.New(futureTime),
				To:              timestamppb.New(pastTime),
				MaxRecords:      100,
				StartFromOldest: false,
			},
			wantErr: true,
		},
		{
			name: "MaxRecords zero",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            timestamppb.New(pastTime),
				To:              timestamppb.New(now),
				MaxRecords:      0,
				StartFromOldest: false,
			},
			wantErr: false,
		},
		{
			name: "MaxRecords negative",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            timestamppb.New(pastTime),
				To:              timestamppb.New(now),
				MaxRecords:      -10,
				StartFromOldest: false,
			},
			wantErr: true,
		},
		{
			name: "Valid request with MaxRecords",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            timestamppb.New(pastTime),
				To:              timestamppb.New(now),
				MaxRecords:      100,
				StartFromOldest: false,
			},
			wantErr: false,
		},
		{
			name: "Valid request without MaxRecords",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            timestamppb.New(pastTime),
				To:              timestamppb.New(now),
				MaxRecords:      0,
				StartFromOldest: false,
			},
			wantErr: false,
		},
		{
			name: "Valid request with equal timestamps",
			req: &GetLogRecordsRequest{
				Scope:           "valid-scope",
				Query:           "valid-query",
				From:            timestamppb.New(now),
				To:              timestamppb.New(now),
				MaxRecords:      100,
				StartFromOldest: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLogRecordsRequest.Validate() error = %v, wantErr %v\nrequest = %v", err, tt.wantErr, tt.req)
			}
		})
	}
}
