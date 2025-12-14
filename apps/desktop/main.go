package main

import (
	"embed"
	_ "embed"
	"log"
	"path/filepath"
	"time"

	"github.com/atticus6/echPlus/apps/desktop/config"
	"github.com/atticus6/echPlus/apps/desktop/database"
	"github.com/atticus6/echPlus/apps/desktop/logger"
	"github.com/atticus6/echPlus/apps/desktop/services"
	"github.com/atticus6/echPlus/apps/desktop/views"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	dbPath := filepath.Join(config.StoreDir, "db.db")

	// 初始化日志系统（按日期和类型拆分）
	logDir := filepath.Join(config.StoreDir, "logs")
	if err := logger.Init(logDir); err != nil {
		log.Fatal("无法初始化日志系统:", err)
	}
	defer logger.Close()

	logger.Info("应用启动，数据库路径: %s", dbPath)

	if err := database.Init(dbPath); err != nil {
		logger.Fatal("数据库初始化失败: %v", err)
	}
	logger.Info("数据库初始化成功")

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	views.MainView = application.New(application.Options{
		Name:        "desktop",
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(&GreetService{}),
			application.NewService(&services.UserService{}),
			application.NewService(&services.NodeService{}),
			application.NewService(&services.ProxyServerInstance),
			application.NewService(&services.ConfigService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			logger.Info("应用程序正在关闭...")
			// 保存配置到文件
			if err := config.ConfigState.SaveConfig(); err != nil {
				logger.Error("保存配置失败: %s", err.Error())
			}
			err := services.ProxyServerInstance.Stop()
			if err != nil {
				logger.Error("%s", err.Error())
			}
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	views.MainView.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Window 1",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			views.MainView.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	// Run the application. This blocks until the application has been exited.
	err := views.MainView.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
