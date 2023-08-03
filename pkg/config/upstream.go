package config

import (
	"strconv"
	"strings"
	"time"
)

type UpstreamConfig struct {
	URL     string                `yaml:"url"`
	Headers map[string]string     `yaml:"headers"`
	FailOn  *UpstreamFailOnConfig `yaml:"fail-on"`
}

type HTTPStatusConfig struct {
	Exact int    `yaml:"exact"`
	Range string `yaml:"range"`
}

func (s *HTTPStatusConfig) Is(status int) bool {
	if s.Exact == status {
		return true
	}

	rangeParts := strings.Split(s.Range, "-")
	if len(rangeParts) != 2 {
		return false
	}

	lower, err1 := strconv.Atoi(rangeParts[0])
	upper, err2 := strconv.Atoi(rangeParts[1])
	if err1 == nil && err2 == nil && status >= lower && status <= upper {
		return true
	}

	return false
}

type HttpStatusFailOnConfig []HTTPStatusConfig

func (ss HttpStatusFailOnConfig) Is(status int) bool {
	for _, s := range ss {
		if s.Is(status) {
			return true
		}
	}

	return false
}

type UpstreamFailOnConfig struct {
	TimeOut    time.Duration          `yaml:"timeout"`
	HTTPStatus HttpStatusFailOnConfig `yaml:"http-status"`
}
