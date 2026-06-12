package service

import (
	"context"

	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
)

var startLLMPodStatusWatcher = models.StartLLMPodStatusWatcher

func startBackgroundWorkers(ctx context.Context, opts *options.LLMOptions) {
	if opts == nil || opts.IsSlaveNode {
		return
	}
	startLLMPodStatusWatcher(ctx, opts.Region)
}
