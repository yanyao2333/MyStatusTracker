package main

import (
	"fmt"
	"net/http"
	"strings"
)

func sendApplicationStatus(appName string, message string, apiEndpoint string, password string) error {
	req, err := http.NewRequest("POST", apiEndpoint+"/update-software", strings.NewReader(fmt.Sprintf(`{"software": "%s", "message": "%s"}`, appName, message)))
	if err != nil {
		return fmt.Errorf("无法创建请求: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Password", password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("无法发送请求: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败: %s", resp.Status)
	}

	return nil
}
