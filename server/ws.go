package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSMessage struct {
	Type     string `json:"type"` // "progress" | "task_complete" | "task_failed"
	Tid      int    `json:"tid"`
	TaskType string `json:"taskType"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`

	CurrentPage  int    `json:"currentPage,omitempty"`
	TotalPage    int    `json:"totalPage,omitempty"`
	CurrentFloor int    `json:"currentFloor,omitempty"`
	TotalFloor   int    `json:"totalFloor,omitempty"`
	Stage        string `json:"stage,omitempty"`

	// 单次下载页数限制（#56）相关信息，用于向前端说明本次并未下载完整
	Limited           bool `json:"limited,omitempty"`
	WebTotalPage      int  `json:"webTotalPage,omitempty"`
	PageDownloadLimit int  `json:"pageDownloadLimit,omitempty"`
}

type WebSocketHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *WebSocketHub) AddClient(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	log.Printf("WebSocket 客户端连接: %s", conn.RemoteAddr().String())
}

func (h *WebSocketHub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
	log.Printf("WebSocket 客户端断开: %s", conn.RemoteAddr().String())
}

func (h *WebSocketHub) Broadcast(msg WSMessage) {
	data, _ := json.Marshal(msg)
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			// 连接已断开，从列表中移除
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *WebSocketHub) BroadcastProgress(task *TaskStatus) {
	msg := WSMessage{
		Type:              "progress",
		Tid:               task.Tid,
		TaskType:          task.Type,
		Status:            task.Status,
		CurrentPage:       task.CurrentPage,
		TotalPage:         task.TotalPage,
		CurrentFloor:      task.CurrentFloor,
		TotalFloor:        task.TotalFloor,
		Stage:             task.Stage,
		Limited:           task.Limited,
		WebTotalPage:      task.WebTotalPage,
		PageDownloadLimit: task.PageDownloadLimit,
	}
	h.Broadcast(msg)
}

func (h *WebSocketHub) BroadcastTaskComplete(task *TaskStatus) {
	msg := WSMessage{
		Type:              "task_complete",
		Tid:               task.Tid,
		TaskType:          task.Type,
		Status:            "completed",
		Limited:           task.Limited,
		WebTotalPage:      task.WebTotalPage,
		PageDownloadLimit: task.PageDownloadLimit,
	}
	if task.Limited {
		// 达到单次下载上限时任务本身是成功的，但帖子并没有下载完整，
		// 必须明确告知用户"还要再来一次"，否则会被误认为已全部导出。
		msg.Message = fmt.Sprintf("已达单次下载页数上限（%d 页）：本次已下载至第 %d 页，全帖共 %d 页，尚未下载完整。请再次执行增量更新以继续下载后续页面。",
			task.PageDownloadLimit, task.TotalPage, task.WebTotalPage)
	} else {
		msg.Message = fmt.Sprintf("下载完成，共 %d 页 %d 楼", task.TotalPage, task.TotalFloor)
	}
	h.Broadcast(msg)
}

func (h *WebSocketHub) BroadcastTaskFailed(task *TaskStatus) {
	msg := WSMessage{
		Type:     "task_failed",
		Tid:      task.Tid,
		TaskType: task.Type,
		Status:   "failed",
		Error:    task.Error,
	}
	h.Broadcast(msg)
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request, password string, hub *WebSocketHub, sessionManager *SessionManager) {
	// 1. 检查 session cookie（浏览器客户端）
	if sessionManager.GetSessionFromRequest(r) != nil {
		// 已通过 cookie 认证
	} else {
		// 2. 回退到 URL query 参数 ?token=base64(admin:password)
		token := r.URL.Query().Get("token")
		if token != "" {
			decoded, err := base64.StdEncoding.DecodeString(token)
			if err != nil || string(decoded) != "admin:"+password {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		} else {
			http.Error(w, "Missing authentication token", http.StatusUnauthorized)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket 升级失败:", err)
		return
	}

	hub.AddClient(conn)

	// 读取循环（保持连接活跃）
	go func() {
		defer hub.RemoveClient(conn)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	// 定期 ping 以保持连接
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hub.mu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			hub.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
}
