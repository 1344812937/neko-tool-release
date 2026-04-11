package tasks

import (
	"neko-tool/pkg/until"

	"github.com/robfig/cron/v3"
)

var log = until.Log

type CronTask struct {
	cron     *cron.Cron
	TaskItem []*CornTaskItem
}

type CornTaskItem struct {
	// 任务名称
	Name string
	// corn 定时任务spec
	Spec string
	// 执行方法
	Handler func()
	// 是否需要首次执行
	FirstExecuted bool
}

// NewCronTask 注册当前定时任务
func NewCronTask() *CronTask {
	res := &CronTask{cron: cron.New()}
	go res.firstExecute()
	return res
}

// 首要任务执行器
func (t *CronTask) firstExecute() {
	item := t.TaskItem
	if item != nil {
		for _, taskItem := range item {
			if taskItem.FirstExecuted {
				t.execute(taskItem)
			}
		}
	}
}

// RegisterTask 任务注册
func (t *CronTask) RegisterTask(taskItem *CornTaskItem) {
	log.Info("注册任务开始：" + taskItem.Name + " spec:" + taskItem.Spec)
	_, err := t.cron.AddFunc(taskItem.Spec, func() {
		t.execute(taskItem)
	})
	if err != nil {
		log.Info("任务注册失败：" + taskItem.Name)
		return
	}
	t.TaskItem = append(t.TaskItem, taskItem)
	log.Info("任务注册成功：" + taskItem.Name)
}

// 定时任务实际执行器
func (t *CronTask) execute(taskItem *CornTaskItem) {
	log.Info("任务调度执行开始：" + taskItem.Name)
	taskItem.Handler()
	log.Info("任务调度执行结束：" + taskItem.Name)
}

func (t *CronTask) Start() {
	t.cron.Start()
	// defer t.cron.Stop()
}

func (t *CronTask) Stop() {
	t.cron.Stop()
}
