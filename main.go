package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/core"
	"github.com/jacopofilonzi/project-library/internal/fsops"
	"github.com/jacopofilonzi/project-library/internal/platform"
	"github.com/jacopofilonzi/project-library/internal/update"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dir, err := config.Dir()
	if err != nil {
		log.Fatal(err)
	}
	store, err := config.Load(filepath.Join(dir, "config.json"), home)
	if err != nil {
		log.Fatal(err)
	}
	lib := core.New(store)
	notifier := notifications.New()

	quitting := false
	app := application.New(application.Options{
		Name:        "Project Library",
		Description: "Browse and open the projects in your Development folder",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(lib),
			application.NewService(notifier),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: projectFiles(lib),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.jacopofilonzi.projectlibrary",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				core.ShowMainWindow()
			},
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	bg := application.NewRGB(0xf4, 0xf4, 0xf1)
	if t := store.Get().Theme; t == "dark" || (t == "system" && app.Env.IsDarkMode()) {
		bg = application.NewRGB(0x12, 0x12, 0x14)
	}
	// "tray only" autostart: the main window starts hidden
	hidden := slices.Contains(os.Args[1:], core.HiddenFlag)
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  "main",
		Title: "Project Library",
		Width: 1280, Height: 800,
		// 2 columns of 220px + a 480px project card
		MinWidth: 920, MinHeight: 600,
		Hidden:           hidden,
		BackgroundColour: bg,
		// Windows: the app header is the title bar (see WindowControls.svelte). Composition hosting
		// lets Wails answer Windows' hit tests from the --wails-non-client-region CSS areas, so the
		// caption buttons behave like native ones (snap layouts, drag, double click to maximize).
		Frameless: platform.CustomTitleBar(),
		Windows: application.WindowsWindow{
			WebView2CompositionHosting: platform.CustomTitleBar(),
		},
		URL: "/",
		// Ctrl/⌘ P does not reach the page (WebView2 reserves it for printing): the window handles it.
		KeyBindings: map[string]func(application.Window){
			"CmdOrCtrl+P": func(application.Window) { app.Event.Emit("shortcut:settings") },
			"CmdOrCtrl+,": func(application.Window) { app.Event.Emit("shortcut:settings") },
		},
	})
	core.MainWindow = win

	// Floating search: borderless window, above the others, not in the taskbar.
	// It hides when it loses focus; the frontend sets its height from the results.
	spot := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "spotlight",
		Title:           "Project Library Search",
		Width:           720,
		Height:          120,
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnFocusLost: true,
		InitialPosition: application.WindowCentered,
		BackgroundType:  application.BackgroundTypeTransparent,
		URL:             "/?view=spotlight",
		Windows: application.WindowsWindow{
			HiddenOnTaskbar:                   true,
			DisableFramelessWindowDecorations: true,
		},
		Mac: application.MacWindow{
			Backdrop:    application.MacBackdropTransparent,
			WindowLevel: application.MacWindowLevelFloating,
		},
	})
	core.SpotlightWindow = spot

	// Closing the window: with "keep in the tray" it hides, otherwise the app quits.
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !quitting && !core.Quitting && store.Get().CloseToTray {
			e.Cancel()
			win.Hide()
			return
		}
		app.Quit()
	})

	lang := func() string { return store.Get().Language }
	menu := application.NewMenu()
	search := menu.Add(trayLabel(lang(), "search"))
	search.OnClick(func(*application.Context) { core.ShowSpotlight() })
	show := menu.Add(trayLabel(lang(), "show"))
	show.OnClick(func(*application.Context) { core.ShowMainWindow() })
	menu.AddSeparator()
	quit := menu.Add(trayLabel(lang(), "quit"))
	quit.OnClick(func(*application.Context) {
		quitting = true
		app.Quit()
	})
	tray := app.SystemTray.New()
	tray.SetIcon(appIcon)
	tray.SetTooltip("Project Library")
	tray.SetMenu(menu)
	tray.OnClick(core.ShowMainWindow)

	// update the tray items when the language changes
	app.Event.On(core.EventConfig, func(*application.CustomEvent) {
		search.SetLabel(trayLabel(lang(), "search"))
		show.SetLabel(trayLabel(lang(), "show"))
		quit.SetLabel(trayLabel(lang(), "quit"))
		menu.Update()
	})

	// new release found at startup: desktop notification, a click brings the app to the front
	core.OnUpdate = func(info update.Info, lang string) {
		err := notifier.SendNotification(notifications.NotificationOptions{
			ID:    "update-" + info.Latest,
			Title: trayLabel(lang, "updateTitle"),
			Body:  strings.ReplaceAll(trayLabel(lang, "updateBody"), "{version}", info.Latest),
		})
		if err != nil {
			log.Printf("update notification: %v", err)
		}
	}
	notifier.OnNotificationResponse(func(notifications.NotificationResult) { core.ShowMainWindow() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func trayLabel(lang, key string) string {
	labels := map[string]map[string]string{
		"en": {"search": "Search projects…", "show": "Show Project Library", "quit": "Quit",
			"updateTitle": "Update available", "updateBody": "Project Library {version} is ready to download."},
		"it": {"search": "Cerca progetti…", "show": "Mostra Project Library", "quit": "Esci",
			"updateTitle": "Aggiornamento disponibile", "updateBody": "Project Library {version} è pronta da scaricare."},
	}
	if l, ok := labels[lang]; ok {
		return l[key]
	}
	return labels["en"][key]
}

var imageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true,
	".webp": true, ".bmp": true, ".ico": true, ".avif": true,
}

// projectFiles serves the images referenced by the READMEs: /project-file?p=<absolute path>.
// Only image files inside the root folders.
func projectFiles(lib *core.Library) application.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/project-file" {
				next.ServeHTTP(w, r)
				return
			}
			p := filepath.Clean(r.URL.Query().Get("p"))
			if !filepath.IsAbs(p) || !imageExt[strings.ToLower(filepath.Ext(p))] || !fsops.Within(p, core.RootsOf(lib)) {
				http.NotFound(w, r)
				return
			}
			// SVGs must not be able to run scripts
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			http.ServeFile(w, r, p)
		})
	}
}
