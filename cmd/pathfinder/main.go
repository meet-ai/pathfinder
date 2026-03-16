package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	var message string
	flag.StringVar(&message, "m", "", "任务描述（目标）")
	flag.Parse()

	if message == "" {
		fmt.Fprintln(os.Stderr, "用法: pathfinder -m \"任务描述\"")
		flag.Usage()
		os.Exit(1)
	}

	daemonURL := os.Getenv("PATHFINDER_DAEMON_URL")
	if daemonURL == "" {
		daemonURL = "http://127.0.0.1:8080"
	}

	reqBody := struct {
		Goal string `json:"goal"`
	}{
		Goal: message,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintln(os.Stderr, "编码请求失败:", err)
		os.Exit(1)
	}

	resp, err := http.Post(daemonURL+"/runs", "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "请求 daemon 失败:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintln(os.Stderr, "daemon 返回错误状态:", resp.Status)
		os.Exit(1)
	}

	var respBody struct {
		URL   string `json:"url"`
		RunID string `json:"runId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		fmt.Fprintln(os.Stderr, "解析 daemon 响应失败:", err)
		os.Exit(1)
	}

	if respBody.URL == "" {
		fmt.Fprintln(os.Stderr, "daemon 响应缺少 url")
		os.Exit(1)
	}

	fmt.Println(respBody.URL)
}
