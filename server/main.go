package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var (
	mu           sync.Mutex
	currentState = State{
		Status:   "离线",
		Software: "",
	}
)

type State struct {
	Status     string    `json:"status"`
	StatusCode int       `json:"status_code"` // 状态标示（1 为在线、2 为离线）
	Software   string    `json:"software"`
	Message    string    `json:"message"` // 直接将状态信息拼接为字符串，方便前端显示
	Timestamp  time.Time `json:"timestamp"`
}

type UpdateStatusRequest struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

type UpdateSoftwareRequest struct {
	Software string `json:"software"`
	Message  string `json:"message"`
}

func (s State) MarshalJSON() ([]byte, error) {
	type Alias State
	return json.Marshal(&struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: s.Timestamp.Unix(),
		Alias:     (*Alias)(&s),
	})
}

func (s *State) UnmarshalJSON(data []byte) error {
	type Alias State
	aux := &struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	s.Timestamp = time.Unix(aux.Timestamp, 0)
	return nil
}

// 如果请求 api 时未提供 message 字段，就生成一个
func (s *State) BuildMessage(message string) {
	// if s.StatusCode == 1 {
	if message != "" {
		s.Message = fmt.Sprintf(`%s\n当前状态：「%s」`, message, s.Status)
		return
	}
	s.Message = fmt.Sprintf(`正在使用「%s」\n当前状态：「%s」`, s.Software, s.Status)
	// } else {
	// 	if message != "" {
	// 		s.Message = fmt.Sprintf(`当前状态：「%s」\n%s\n等会再找我吧！`, s.Status, message)
	// 		return
	// 	}
	// 	s.Message = fmt.Sprintf(`当前状态：「%s」\n正在使用「%s」\n等会再找我吧！`, s.Status, s.Software)
	// }
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("无法加载环境变量文件: %s\n", err)
		return
	}
	log.Println("环境变量加载成功.")
	log.Println("配置文件加载成功")

	if err := loadState(); err != nil {
		log.Printf("无法加载状态文件，将创建新文件: %v", err)
	} else {
		log.Println("状态文件加载成功")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/events", sseHandler)
	mux.HandleFunc("/update-status", authMiddleware(updateStatusHandler))
	mux.HandleFunc("/update-software", authMiddleware(updateSoftwareHandler))

	// 添加CORS中间件
	handler := corsMiddleware(mux)

	log.Println("服务器启动在 :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// 鉴权中间件
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedPassword := r.Header.Get("X-Password")
		if providedPassword != os.Getenv("API_PASSWORD") {
			log.Printf("鉴权失败，请求来自 %s", r.RemoteAddr)
			http.Error(w, "未授权的访问", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Printf("新的 SSE 客户端连接来自: %s", r.RemoteAddr)

	for {
		select {
		case <-ticker.C:
			mu.Lock()
			data, _ := currentState.MarshalJSON()
			mu.Unlock()

			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()

		case <-r.Context().Done():
			log.Printf("SSE 客户端断开连接来自: %s", r.RemoteAddr)
			return
		}
	}
}

func updateStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("更新状态请求解析失败，请求来自 %s, 错误: %v", r.RemoteAddr, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	currentState.Status = req.Status
	currentState.StatusCode = req.StatusCode
	currentState.BuildMessage(req.Message)
	currentState.Timestamp = time.Now()
	mu.Unlock()

	if err := saveState(); err != nil {
		log.Printf("保存状态失败, 错误: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("更新状态成功，请求来自: %s, 状态: %s, 状态码: %d", r.RemoteAddr, req.Status, req.StatusCode)
	w.WriteHeader(http.StatusOK)
}

func updateSoftwareHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateSoftwareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("更新软件请求解析失败，请求来自 %s, 错误: %v", r.RemoteAddr, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	currentState.Software = req.Software
	currentState.Timestamp = time.Now()
	currentState.BuildMessage(req.Message)
	mu.Unlock()

	if err := saveState(); err != nil {
		log.Printf("保存状态失败, 错误: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("更新软件成功，请求来自: %s, 软件: %s", r.RemoteAddr, req.Software)
	w.WriteHeader(http.StatusOK)
}

func saveState() error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Create("state.json")
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := currentState.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = file.Write(data)

	return err
}

func loadState() error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Open("state.json")
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	return currentState.UnmarshalJSON(data)
}

// 修改CORS中间件以支持鉴权头
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Password")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}
