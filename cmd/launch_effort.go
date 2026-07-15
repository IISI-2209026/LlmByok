package cmd

import (
	"fmt"
	"strings"
)

var launchEffortLevels = map[string][]string{
	"copilot":   {"none", "minimal", "low", "medium", "high", "xhigh", "max"},
	"codex":     {"none", "minimal", "low", "medium", "high", "xhigh", "max"},
	"codex-app": {"none", "minimal", "low", "medium", "high", "xhigh", "max"},
	"claude":    {"low", "medium", "high", "xhigh", "max"},
	"pi":        {"off", "minimal", "low", "medium", "high", "xhigh", "max"},
}

func validateLaunchEffort(target, effort string) error {
	if effort == "" {
		return nil
	}
	for _, level := range launchEffortLevels[target] {
		if level == effort {
			return nil
		}
	}
	return fmt.Errorf("target %s 不支援 effort %q；有效值：%s", target, effort, strings.Join(launchEffortLevels[target], ", "))
}
