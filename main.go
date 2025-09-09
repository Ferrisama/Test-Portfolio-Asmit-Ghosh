package main

import (
	"embed"
	_ "embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"changeme/backend/bindings"
	"changeme/backend/repo"

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

	// Predeclare services to register via application.Options
	js := &bindings.JournalService{Repo: nil, AppVersion: "0.1.0"}

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Services' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "portfolio",
		Description: "Trading Journal - A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(js),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Initialize backend services & repo
	dbDir := "data"
	_ = os.MkdirAll(dbDir, 0o755)
	dbPath := filepath.Join(dbDir, "app.db")
	sqliteRepo, err := repo.NewSQLiteRepo(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := repo.ApplyMigrations(sqliteRepo.DB, "migrations"); err != nil {
		log.Fatal(err)
	}

	// Wire the initialized repository into the pre-registered JournalService
	js.Repo = sqliteRepo

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	_ = app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "Trading Journal",
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
			app.EmitEvent("time", now)
			time.Sleep(time.Second)
		}
	}()

	// Run the application. This blocks until the application has been exited.
	err = app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}