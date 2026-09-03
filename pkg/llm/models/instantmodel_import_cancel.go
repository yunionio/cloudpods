package models

import (
	"context"
	"sync"
)

var instantModelImportCancels sync.Map // map[string]context.CancelFunc

// beginImportContext registers a cancel func so StartDeleteTask can abort an in-flight download.
func (model *SInstantModel) beginImportContext(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	modelId := model.GetId()
	instantModelImportCancels.Store(modelId, cancel)
	return ctx, func() {
		cancel()
		instantModelImportCancels.Delete(modelId)
	}
}

// CancelInstantModelImport cancels an in-flight DoImport for the given model id, if any.
func CancelInstantModelImport(modelId string) {
	if modelId == "" {
		return
	}
	if v, ok := instantModelImportCancels.LoadAndDelete(modelId); ok {
		if cancel, ok := v.(context.CancelFunc); ok && cancel != nil {
			cancel()
		}
	}
}
