package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func getWindowProperties(windowID string) (wmClass []string, wmName string, err error) {
	cmd := exec.Command("xprop", "-id", windowID)
	output, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("无法创建管道: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("无法执行命令: %w", err)
	}

	defer func() {
		_ = cmd.Wait()
		_ = output.Close()
	}()

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()

		// 解析 WM_CLASS
		if strings.HasPrefix(line, "WM_CLASS(STRING) = ") {
			parts := strings.SplitN(line, "= ", 2)
			if len(parts) > 1 {
				classStr := parts[1]
				re := regexp.MustCompile(`"([^"]*)"`)
				matches := re.FindAllStringSubmatch(classStr, -1)
				for _, match := range matches {
					if len(match) > 1 {
						wmClass = append(wmClass, match[1])
					}
				}

			}
		}
		// 解析 WM_NAME
		if strings.HasPrefix(line, "WM_NAME(UTF8_STRING) = ") || strings.HasPrefix(line, "WM_NAME(COMPOUND_TEXT) = ") {
			parts := strings.SplitN(line, "= ", 2)
			if len(parts) > 1 {

				wmName = strings.Trim(parts[1], "\"")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("读取 xprop 输出失败: %w", err)
	}

	return wmClass, wmName, nil
}

func getActiveWindowID() (string, error) {
	cmd := exec.Command("xdotool", "getactivewindow")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("无法执行 xdotool getactivewindow: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
