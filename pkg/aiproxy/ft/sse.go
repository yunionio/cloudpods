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

package ft

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

func openAIChoiceDelta(obj jsonutils.JSONObject, field string) (string, error) {
	arr, err := obj.GetArray("choices")
	if err != nil || len(arr) == 0 {
		return "", err
	}
	if field == "delta" {
		return arr[0].GetString("delta", "content")
	}
	return arr[0].GetString("message", "content")
}

func parseOpenAIStreamDelta(payload string) (string, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" || payload == "[DONE]" {
		return "", nil
	}
	obj, err := jsonutils.ParseString(payload)
	if err != nil {
		return "", err
	}
	if obj.Contains("error") {
		return "", errors.Errorf("stream error event: %s", payload)
	}
	return openAIChoiceDelta(obj, "delta")
}

func parseAnthropicStreamDelta(payload string) (string, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" || payload == "[DONE]" {
		return "", nil
	}
	obj, err := jsonutils.ParseString(payload)
	if err != nil {
		return "", err
	}
	delta, _ := obj.GetString("delta", "text")
	return delta, nil
}

func aggregateSSEStream(r io.Reader, parseDelta func(string) (string, error)) (string, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	var aggregated strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		delta, err := parseDelta(payload)
		if err != nil {
			return "", err
		}
		aggregated.WriteString(delta)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	out := aggregated.String()
	if out == "" {
		return "", errors.Error("empty aggregated stream content")
	}
	return out, nil
}

func extractOpenAIChatContent(body []byte) (string, error) {
	obj, err := jsonutils.Parse(body)
	if err != nil {
		return "", err
	}
	content, err := openAIChoiceDelta(obj, "message")
	if err != nil {
		return "", errors.Wrap(err, "parse choices")
	}
	if content == "" {
		return "", errors.Errorf("empty choices[0].message.content: %s", truncateBody(body))
	}
	return content, nil
}

func extractAnthropicTextContent(body []byte) (string, error) {
	obj, err := jsonutils.Parse(body)
	if err != nil {
		return "", err
	}
	arr, err := obj.GetArray("content")
	if err != nil {
		return "", errors.Wrap(err, "parse content array")
	}
	for _, block := range arr {
		typ, _ := block.GetString("type")
		if typ == "text" {
			text, _ := block.GetString("text")
			if text != "" {
				return text, nil
			}
		}
	}
	return "", errors.Errorf("empty anthropic text content block: %s", truncateBody(body))
}

func truncateBody(body []byte) string {
	const max = 512
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "..."
}

func printJSONBody(body []byte) error {
	obj, err := jsonutils.Parse(body)
	if err != nil {
		fmt.Println(string(body))
		return nil
	}
	fmt.Println(obj.PrettyString())
	return nil
}

func previewText(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func dumpStreamPreview(body []byte) {
	lines := bytes.Split(body, []byte("\n"))
	limit := 40
	if len(lines) < limit {
		limit = len(lines)
	}
	fmt.Println("--- stream body (first lines) ---")
	for i := 0; i < limit; i++ {
		fmt.Println(string(lines[i]))
	}
}
