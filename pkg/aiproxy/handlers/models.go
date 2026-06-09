// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

// modelsHandler implements OpenAI-compatible GET /openai/v1/models.
func modelsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httperrors.InvalidInputError(ctx, w, "only GET is supported")
		return
	}
	vk := extractVirtualKey(r)
	userCred := auth.AdminCredential()
	items, err := models.ListModelsForVirtualKey(ctx, userCred, vk)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	if items == nil {
		items = []models.ModelsListEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   items,
	})
}

// modelRetrieveHandler implements OpenAI-compatible GET /openai/v1/models/{model}.
func modelRetrieveHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httperrors.InvalidInputError(ctx, w, "only GET is supported")
		return
	}
	params := appsrv.AppContextGetParams(ctx)
	modelID := ""
	if params != nil {
		modelID = strings.TrimSpace(params.Params["<model>"])
	}
	if modelID == "" {
		httperrors.InvalidInputError(ctx, w, "missing model id")
		return
	}
	vk := extractVirtualKey(r)
	userCred := auth.AdminCredential()
	items, err := models.ListModelsForVirtualKey(ctx, userCred, vk)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	for _, item := range items {
		if item.ID == modelID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(item)
			return
		}
	}
	httperrors.NotFoundError(ctx, w, "model %q not found", modelID)
}
