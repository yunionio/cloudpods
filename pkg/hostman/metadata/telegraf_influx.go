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
// See the License for the specific

package metadata

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
)

const telegrafInfluxMaxBodyBytes = 16 << 20

func (s *Service) rewriteTelegrafInfluxBodyIfNeeded(ctx context.Context, r *http.Request) error {
	prefix := s.monitorPrefix()
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return nil
	}
	sub := r.URL.Path[len(prefix):]
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		return nil
	}
	if sub != "/write" && !strings.HasPrefix(sub, "/write?") {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, telegrafInfluxMaxBodyBytes+1))
	if err != nil {
		return errors.Wrap(err, "read telegraf influx body")
	}
	if len(body) > telegrafInfluxMaxBodyBytes {
		return errors.Errorf("telegraf influx body exceeds %d bytes", telegrafInfluxMaxBodyBytes)
	}
	_ = r.Body.Close()

	if strings.Contains(strings.ToLower(r.Header.Get("Content-Encoding")), "gzip") {
		gr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return errors.Wrap(err, "gzip reader")
		}
		body, err = io.ReadAll(io.LimitReader(gr, telegrafInfluxMaxBodyBytes+1))
		_ = gr.Close()
		if err != nil {
			return errors.Wrap(err, "read gzipped telegraf body")
		}
		if len(body) > telegrafInfluxMaxBodyBytes {
			return errors.Errorf("telegraf influx body exceeds %d bytes after gzip", telegrafInfluxMaxBodyBytes)
		}
		r.Header.Del("Content-Encoding")
	}

	newBody, changed, err := rewriteInfluxLineProtocolTags(body, func(vmId string) (map[string]string, bool) {
		gd := s.lookupGuestDescForTelegraf(r, vmId)
		if gd == nil {
			return nil, false
		}
		tags := make(map[string]string)
		if gd.TenantId != "" {
			tags["tenant_id"] = gd.TenantId
		}
		if gd.Name != "" {
			tags["vm_name"] = gd.Name
		}
		if len(tags) == 0 {
			return nil, false
		}
		return tags, true
	})
	if err != nil {
		return err
	}
	if changed {
		log.Debugf("metadata monitor: corrected telegraf influx tags from %s", r.RemoteAddr)
	}
	r.Body = io.NopCloser(bytes.NewReader(newBody))
	r.ContentLength = int64(len(newBody))
	r.Header.Del("Content-Length")
	return nil
}

func (s *Service) lookupGuestDescForTelegraf(r *http.Request, vmId string) *desc.SGuestDesc {
	if vmId == "" {
		return nil
	}
	gd := s.getGuestDesc(r)
	if gd != nil && gd.Uuid == vmId {
		return gd
	}
	return nil
}

func rewriteInfluxLineProtocolTags(body []byte, resolveTags func(vmId string) (map[string]string, bool)) ([]byte, bool, error) {
	raw := strings.Split(string(body), "\n")
	changed := false
	for i, line := range raw {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		newLine, lineChanged, err := rewriteInfluxLineTags(line, resolveTags)
		if err != nil {
			return body, false, err
		}
		if lineChanged {
			changed = true
			raw[i] = newLine
		}
	}
	if !changed {
		return body, false, nil
	}
	return []byte(strings.Join(raw, "\n")), true, nil
}

func rewriteInfluxLineTags(line string, resolveTags func(vmId string) (map[string]string, bool)) (string, bool, error) {
	measTags, fields, ok := splitMeasurementTagsAndFields(line)
	if !ok {
		return line, false, nil
	}
	parts := splitOnUnescapedComma(measTags)
	if len(parts) < 1 {
		return line, false, nil
	}
	measurement := parts[0]
	tagSegs := parts[1:]
	vmId := ""
	for _, seg := range tagSegs {
		k, v := splitInfluxTagKeyValue(seg)
		if k == "vm_id" {
			vmId = v
			break
		}
	}
	if vmId == "" {
		return line, false, nil
	}
	expectTags, ok := resolveTags(vmId)
	if !ok {
		return line, false, nil
	}
	changed := false
	haveExpect := map[string]bool{}
	newSegs := make([]string, 0, len(tagSegs)+len(expectTags))
	for _, seg := range tagSegs {
		k, v := splitInfluxTagKeyValue(seg)
		expect, isExpect := expectTags[k]
		if isExpect {
			haveExpect[k] = true
			if v != expect {
				changed = true
				newSegs = append(newSegs, k+"="+expect)
			} else {
				newSegs = append(newSegs, seg)
			}
		} else {
			newSegs = append(newSegs, seg)
		}
	}
	for k, v := range expectTags {
		if !haveExpect[k] {
			changed = true
			newSegs = append(newSegs, k+"="+v)
		}
	}
	if !changed {
		return line, false, nil
	}
	var b strings.Builder
	b.WriteString(measurement)
	for _, seg := range newSegs {
		b.WriteByte(',')
		b.WriteString(seg)
	}
	b.WriteByte(' ')
	b.WriteString(fields)
	return b.String(), true, nil
}

// rewriteInfluxLineTenant rewrites tenant_id tag only. Kept for unit tests.
func rewriteInfluxLineTenant(line string, resolveTenant func(vmId string) (tenantId string, ok bool)) (string, bool, error) {
	return rewriteInfluxLineTags(line, func(vmId string) (map[string]string, bool) {
		tenantId, ok := resolveTenant(vmId)
		if !ok {
			return nil, false
		}
		return map[string]string{"tenant_id": tenantId}, true
	})
}

func rewriteInfluxLineProtocolTenant(body []byte, resolveTenant func(vmId string) (tenantId string, ok bool)) ([]byte, bool, error) {
	return rewriteInfluxLineProtocolTags(body, func(vmId string) (map[string]string, bool) {
		tenantId, ok := resolveTenant(vmId)
		if !ok {
			return nil, false
		}
		return map[string]string{"tenant_id": tenantId}, true
	})
}

func splitMeasurementTagsAndFields(line string) (measTags string, fields string, ok bool) {
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' && !influxByteEscaped(line, i) {
			return line[:i], line[i+1:], true
		}
	}
	return "", "", false
}

func influxByteEscaped(line string, i int) bool {
	if i == 0 {
		return false
	}
	n := 0
	for j := i - 1; j >= 0 && line[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

func splitOnUnescapedComma(s string) []string {
	var out []string
	var b strings.Builder
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			b.WriteByte('\\')
			continue
		}
		if c == ',' {
			out = append(out, b.String())
			b.Reset()
			continue
		}
		b.WriteByte(c)
	}
	out = append(out, b.String())
	return out
}

func splitInfluxTagKeyValue(seg string) (key, val string) {
	for i := 0; i < len(seg); i++ {
		if seg[i] == '=' && !influxByteEscaped(seg, i) {
			return influxUnescapeTagKey(seg[:i]), seg[i+1:]
		}
	}
	return "", ""
}

func influxUnescapeTagKey(s string) string {
	return influxUnescapeTag(s)
}

func influxUnescapeTag(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\', ' ', ',', '=':
				b.WriteByte(s[i+1])
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
