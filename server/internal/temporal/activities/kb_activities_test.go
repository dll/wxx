package activities

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
)

func TestKBImportActivity_Success(t *testing.T) {
	acts := &KBActivities{
		ImportResources: func(ndjsonData, username string) (*model.KBImportResponse, error) {
			if username != "admin" {
				t.Errorf("期望 username=admin，得到 %s", username)
			}
			return &model.KBImportResponse{
				Code:    0,
				Message: "导入完成",
				Total:   1,
				Created: 1,
			}, nil
		},
	}

	result, err := acts.KBImportActivity(context.Background(), KBImportInput{
		NDJSONData: `{"resource_id":"test","resource_type":"Policy","title":"测试","content":"正文"}`,
		Username:   "admin",
	})
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if result.ImportResultJSON == "" {
		t.Error("ImportResultJSON 不应为空")
	}
}

func TestKBImportActivity_Error(t *testing.T) {
	acts := &KBActivities{
		ImportResources: func(ndjsonData, username string) (*model.KBImportResponse, error) {
			return nil, context.DeadlineExceeded
		},
	}

	_, err := acts.KBImportActivity(context.Background(), KBImportInput{
		NDJSONData: "bad data",
		Username:   "admin",
	})
	if err == nil {
		t.Error("导入失败应返回错误")
	}
}
