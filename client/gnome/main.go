package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/joho/godotenv"
)

// 自定义系统事件类型
type SystemEvent int

const (
	EventSuspend SystemEvent = iota
	EventResume
	EventShutdown
)

func main() {
	log.Println("程序启动...")
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("无法加载环境变量文件: %s\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 系统事件通道
	sysEvents := make(chan SystemEvent, 10)
	go listenSystemEvents(ctx, sysEvents)

	// 退出信号处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 主循环控制
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	paused := false

	for {
		select {
		case <-ticker.C:
			if !paused {
				updatePipe()
				log.Println("等待下一次更新...")
			}

		case event := <-sysEvents:
			switch event {
			case EventSuspend:
				log.Println("处理休眠事件...")
				sendSuspendStatus()
				paused = true

			case EventResume:
				log.Println("处理恢复事件...")
				paused = false
				updatePipe() // 立即更新一次
				log.Println("恢复定期更新...")

			case EventShutdown:
				log.Println("处理关机事件...")
				sendShutdownStatus()
				cancel()
				return
			}

		case <-quit:
			log.Println("收到退出信号，停止程序...")
			cancel()
			return
		}
	}
}

func listenSystemEvents(ctx context.Context, notifyChan chan<- SystemEvent) {
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Printf("无法连接系统总线: %v\n", err)
		return
	}
	defer conn.Close()

	// 匹配系统事件信号
	matchRules := []dbus.MatchOption{
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
	}

	if err := conn.AddMatchSignal(matchRules...); err != nil {
		log.Printf("添加信号匹配失败: %v\n", err)
		return
	}

	sigChan := make(chan *dbus.Signal, 10)
	conn.Signal(sigChan)

	for {
		select {
		case sig := <-sigChan:
			log.Printf("捕获到信号: %v\n", sig)
			switch sig.Name {
			case "org.freedesktop.login1.Manager.PrepareForSleep":
				if entering, ok := sig.Body[0].(bool); ok {
					if entering {
						log.Println("捕获到休眠开始信号")
						notifyChan <- EventSuspend
					} else {
						log.Println("捕获到休眠恢复信号")
						notifyChan <- EventResume
					}
				}

			case "org.freedesktop.login1.Manager.PrepareForShutdown":
				if entering, ok := sig.Body[0].(bool); ok && entering {
					log.Println("捕获到关机信号")
					notifyChan <- EventShutdown
				}
			}

		case <-ctx.Done():
			log.Println("停止系统事件监听...")
			return
		}
	}
}

// 发送关机状态
func sendShutdownStatus() {
	err := sendApplicationStatus("system", "电脑已关机💀", os.Getenv("API_ENDPOINT"), os.Getenv("PASSWORD"))
	if err != nil {
		log.Printf("发送关机状态失败: %v\n", err)
	} else {
		log.Println("关机状态已发送")
	}
}

func sendSuspendStatus() {
	err := sendApplicationStatus("system", "电脑休眠中💤", os.Getenv("API_ENDPOINT"), os.Getenv("PASSWORD"))
	if err != nil {
		log.Printf("发送休眠状态失败: %v\n", err)
	} else {
		log.Println("休眠状态已发送")
	}
}

func getWindowProp() ([]string, string, error) {
	log.Println("获取窗口属性...")
	windowID, err := getActiveWindowID()
	if err != nil {
		log.Printf("获取活动窗口ID失败: %v\n", err)
		return nil, "", err
	}
	log.Printf("活动窗口ID: %s\n", windowID)

	wmClass, wmName, err := getWindowProperties(windowID)
	if err != nil {
		log.Printf("获取窗口属性失败: %v\n", err)
		return nil, "", err
	}
	log.Printf("窗口类名: %v, 窗口名称: %s\n", wmClass, wmName)

	return wmClass, wmName, nil
}

func updatePipe() {
	wmClass, wmName, err := getWindowProp()
	if err != nil {
		log.Println("获取窗口属性失败，跳过更新管道.")
		return
	}

	appName, message := matchApplicationPatterns(wmName, wmClass[len(wmClass)-1])
	log.Printf("正在发送应用状态，应用名: %s, 生成消息: %s\n", appName, message)
	err = sendApplicationStatus(appName, message, os.Getenv("API_ENDPOINT"), os.Getenv("PASSWORD"))
	if err != nil {
		log.Printf("发送应用状态失败: %v\n", err)
	} else {
		log.Println("成功发送应用状态.")
	}
}
