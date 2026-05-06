package temporal

import (
	"log"

	"github.com/dll/wxx/server/internal/temporal/activities"
	"github.com/dll/wxx/server/internal/temporal/workflows"
	"go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
)

// Activities 聚合所有活动结构体（供 main.go 构造和注入）
type Activities struct {
	Chat        *activities.ChatActivities
	Emotion     *activities.EmotionActivities
	Integration *activities.IntegrationActivities
	KB          *activities.KBActivities
}

// StartWorker 启动 Temporal worker，注册所有工作流和活动
// 返回 Worker 实例供调用方管理生命周期
func StartWorker(c client.Client, taskQueue string, acts *Activities) sdkworker.Worker {
	w := sdkworker.New(c, taskQueue, sdkworker.Options{})

	// 注册工作流
	w.RegisterWorkflow(workflows.ChatAskWorkflow)
	w.RegisterWorkflow(workflows.EmotionAnalyzeWorkflow)
	w.RegisterWorkflow(workflows.IntegrationProxyWorkflow)
	w.RegisterWorkflow(workflows.KBImportWorkflow)

	// 注册活动
	w.RegisterActivity(acts.Chat)
	w.RegisterActivity(acts.Emotion)
	w.RegisterActivity(acts.Integration)
	w.RegisterActivity(acts.KB)

	// 非阻塞启动 worker
	go func() {
		if err := w.Run(sdkworker.InterruptCh()); err != nil {
			log.Printf("Temporal worker 异常退出: %v", err)
		}
	}()

	log.Printf("Temporal worker 已启动: task_queue=%s", taskQueue)
	return w
}
