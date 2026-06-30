// Package user 包含用户服务的单元测试。
// GetUser 优先查 MySQL，未命中时走 mock 兜底。
package user

import (
	"context"
	"testing"

	userpb "xys-clone/proto/user"
)

// TestGetUserMock 验证无 DB 连接时（纯 mock）GetUser 返回各字段正确。
func TestGetUserMock(t *testing.T) {
	s := NewServer(nil, nil) // db = nil → 纯 mock

	resp, err := s.GetUser(context.Background(), &userpb.GetUserReq{UserId: 1})
	if err != nil {
		t.Fatalf("GetUser 返回错误: %v", err)
	}

	if resp.Id != 1 {
		t.Errorf("期望 Id = 1，实际 = %d", resp.Id)
	}
	if resp.Username != "mock_user" {
		t.Errorf("期望 Username = mock_user，实际 = %s", resp.Username)
	}
	if resp.Bio == "" {
		t.Error("Bio 不应为空")
	}
	if resp.Avatar == "" {
		t.Error("Avatar 不应为空")
	}
	if resp.CreatedAt == 0 {
		t.Error("CreatedAt 不应为 0")
	}

	t.Logf("GetUser(1) => id=%d username=%s bio=%s", resp.Id, resp.Username, resp.Bio)
}
