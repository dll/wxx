package activities

import (
	"context"
	"encoding/json"

	"github.com/dll/wxx/server/internal/model"
)

// KBActivities 知识库相关活动集合
// 使用函数字段注入而非直接引用 service 包（避免循环依赖）
type KBActivities struct {
	// ImportResources 批量导入知识资源
	ImportResources func(ndjsonData, username string) (*model.KBImportResponse, error)
}

// KBImportInput 知识导入活动输入
type KBImportInput struct {
	NDJSONData string `json:"ndjson_data"`
	Username   string `json:"username"`
}

// KBImportOutput 知识导入活动输出
type KBImportOutput struct {
	ImportResultJSON string `json:"import_result_json"`
}

// KBImportActivity 批量导入知识资源
func (a *KBActivities) KBImportActivity(ctx context.Context, input KBImportInput) (*KBImportOutput, error) {
	resp, err := a.ImportResources(input.NDJSONData, input.Username)
	if err != nil {
		return nil, err
	}

	respJSON, _ := json.Marshal(resp)
	return &KBImportOutput{ImportResultJSON: string(respJSON)}, nil
}
