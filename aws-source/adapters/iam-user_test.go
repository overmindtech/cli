package adapters

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func (t *TestIAMClient) ListGroupsForUser(ctx context.Context, params *iam.ListGroupsForUserInput, optFns ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error) {
	isTruncated := true
	marker := params.Marker

	if marker == nil {
		marker = PtrString("0")
	}

	// Get the current page
	markerInt, _ := strconv.Atoi(*marker)

	// Set the marker to the next page
	markerInt++

	if markerInt >= 3 {
		isTruncated = false
		marker = nil
	} else {
		marker = PtrString(fmt.Sprint(markerInt))
	}

	return &iam.ListGroupsForUserOutput{
		Groups: []types.Group{
			{
				Arn:        PtrString("arn:aws:iam::801795385023:Group/something"),
				CreateDate: PtrTime(time.Now()),
				GroupId:    PtrString("id"),
				GroupName:  PtrString(fmt.Sprintf("group-%v", marker)),
				Path:       PtrString("/"),
			},
		},
		IsTruncated: isTruncated,
		Marker:      marker,
	}, nil
}

func (t *TestIAMClient) GetUser(ctx context.Context, params *iam.GetUserInput, optFns ...func(*iam.Options)) (*iam.GetUserOutput, error) {
	return &iam.GetUserOutput{
		User: &types.User{
			Path:       PtrString("/"),
			UserName:   PtrString("power-users"),
			UserId:     PtrString("AGPA3VLV2U27T6SSLJMDS"),
			Arn:        PtrString("arn:aws:iam::801795385023:User/power-users"),
			CreateDate: PtrTime(time.Now()),
		},
	}, nil
}

func (t *TestIAMClient) ListUsers(ctx context.Context, params *iam.ListUsersInput, optFns ...func(*iam.Options)) (*iam.ListUsersOutput, error) {
	isTruncated := true
	marker := params.Marker

	if marker == nil {
		marker = PtrString("0")
	}

	// Get the current page
	markerInt, _ := strconv.Atoi(*marker)

	// Set the marker to the next page
	markerInt++

	if markerInt >= 3 {
		isTruncated = false
		marker = nil
	} else {
		marker = PtrString(fmt.Sprint(markerInt))
	}

	return &iam.ListUsersOutput{
		Users: []types.User{
			{
				Path:       PtrString("/"),
				UserName:   PtrString(fmt.Sprintf("user-%v", marker)),
				UserId:     PtrString("AGPA3VLV2U27T6SSLJMDS"),
				Arn:        PtrString("arn:aws:iam::801795385023:User/power-users"),
				CreateDate: PtrTime(time.Now()),
			},
		},
		IsTruncated: isTruncated,
		Marker:      marker,
	}, nil
}

func (t *TestIAMClient) ListUserTags(context.Context, *iam.ListUserTagsInput, ...func(*iam.Options)) (*iam.ListUserTagsOutput, error) {
	return &iam.ListUserTagsOutput{
		Tags: []types.Tag{
			{
				Key:   PtrString("foo"),
				Value: PtrString("bar"),
			},
		},
		IsTruncated: false,
		Marker:      nil,
	}, nil
}

func TestGetUserGroups(t *testing.T) {
	groups, err := getUserGroups(context.Background(), &TestIAMClient{}, PtrString("foo"))
	if err != nil {
		t.Error(err)
	}

	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %v", len(groups))
	}
}

func TestUserGetFunc(t *testing.T) {
	user, err := userGetFunc(context.Background(), &TestIAMClient{}, "foo", "bar")
	if err != nil {
		t.Error(err)
	}

	if user.User == nil {
		t.Error("user is nil")
	}

	if len(user.UserGroups) != 3 {
		t.Errorf("expected 3 groups, got %v", len(user.UserGroups))
	}
}

func TestUserListFunc(t *testing.T) {
	adapter := NewIAMUserAdapter(&TestIAMClient{}, "foo", nil)

	stream := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(context.Background(), "foo", false, stream)

	errs := stream.GetErrors()
	if len(errs) > 0 {
		t.Error(errs)
	}

	items := stream.GetItems()
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %v", len(items))
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
		if len(item.GetLinkedItemQueries()) != 3 {
			t.Errorf("expected 3 linked item queries, got %v", len(item.GetLinkedItemQueries()))
		}
	}
}

func TestUserListTagsFunc(t *testing.T) {
	tags, err := userListTagsFunc(context.Background(), &UserDetails{
		User: &types.User{
			UserName: PtrString("foo"),
		},
	}, &TestIAMClient{})
	if err != nil {
		t.Error(err)
	}

	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %v", len(tags))
	}
}

func TestUserItemMapper(t *testing.T) {
	details := UserDetails{
		User: &types.User{
			Path:       PtrString("/"),
			UserName:   PtrString("power-users"),
			UserId:     PtrString("AGPA3VLV2U27T6SSLJMDS"),
			Arn:        PtrString("arn:aws:iam::801795385023:User/power-users"),
			CreateDate: PtrTime(time.Now()),
		},
		UserGroups: []types.Group{
			{
				Arn:        PtrString("arn:aws:iam::801795385023:Group/something"),
				CreateDate: PtrTime(time.Now()),
				GroupId:    PtrString("id"),
				GroupName:  PtrString("name"),
				Path:       PtrString("/"),
			},
		},
	}

	item, err := userItemMapper(nil, "foo", &details)
	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "iam-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "name",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewIAMUserAdapter(t *testing.T) {
	config, account, _ := GetAutoConfig(t)
	client := iam.NewFromConfig(config, func(o *iam.Options) {
		o.RetryMode = aws.RetryModeAdaptive
		o.RetryMaxAttempts = 10
	})

	adapter := NewIAMUserAdapter(client, account, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 30 * time.Second,
	}

	test.Run(t)
}
