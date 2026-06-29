// Package user 包含用户服务的单元测试。
// 当前针对 mock 实现做基础校验，接入数据库后需扩展测试用例。
package user

import (
	"context"
	"testing"

	userpb "xys-clone/proto/user"
)

// TestGetUser 验证 GetUser mock 返回的各项字段是否正确。
// 测试覆盖：错误为空、返回 id 与入参一致、username / bio / avatar 非空、created_at > 0。
func TestGetUser(t *testing.T) {
	s := NewServer()

	resp, err := s.GetUser(context.Background(), &userpb.GetUserReq{UserId: 1})
	if err != nil {
		t.Fatalf("GetUser 返回错误: %v", err)
	}

	// 校验返回字段
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
