/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"strconv"
	"strings"
)

type Properties struct {
	innerProps map[string]string
}

func NewProperties() *Properties {
	p := Properties{
		innerProps: make(map[string]string, 50),
	}
	return &p
}

func (g *Properties) SetProperties(p *Properties) {
	if p == nil {
		return
	}
	for k, v := range p.innerProps {
		g.Set(strings.ToLower(k), v)
	}
}

func (g *Properties) Len() int {
	return len(g.innerProps)
}

func (g *Properties) IsNil() bool {
	return g == nil || g.innerProps == nil
}

func (g *Properties) GetString(key, def string) string {
	v, ok := g.innerProps[strings.ToLower(key)]

	if !ok || v == "" {
		return def
	}
	return v
}

func (g *Properties) GetInt(key string, def int, min int, max int) int {
	value, ok := g.innerProps[strings.ToLower(key)]
	if !ok || value == "" {
		return def
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return def
	}

	if i > max || i < min {
		return def
	}
	return i
}

func (g *Properties) GetBool(key string, def bool) bool {
	value, ok := g.innerProps[strings.ToLower(key)]
	if !ok || value == "" {
		return def
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return def
	}
	return b
}

func (g *Properties) GetTrimString(key string, def string) string {
	value, ok := g.innerProps[strings.ToLower(key)]
	if !ok || value == "" {
		return def
	} else {
		return strings.TrimSpace(value)
	}
}

func (g *Properties) GetStringArray(key string, def []string) []string {
	value, ok := g.innerProps[strings.ToLower(key)]
	if ok || value != "" {
		array := strings.Split(value, ",")
		if len(array) > 0 {
			return array
		}
	}
	return def
}

//func (g *Properties) GetBool(key string) bool {
//	i, _ := strconv.ParseBool(g.innerProps[key])
//	return i
//}

func (g *Properties) Set(key, value string) {
	g.innerProps[strings.ToLower(key)] = value
}

func (g *Properties) SetIfNotExist(key, value string) {
	if _, ok := g.innerProps[strings.ToLower(key)]; !ok {
		g.Set(key, value)
	}
}

// 如果p有g没有的键值对,添加进g中
func (g *Properties) SetDiffProperties(p *Properties) {
	if p == nil {
		return
	}
	for k, v := range p.innerProps {
		if _, ok := g.innerProps[strings.ToLower(k)]; !ok {
			g.innerProps[strings.ToLower(k)] = v
		}
	}
}
