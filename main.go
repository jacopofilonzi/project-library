package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/core"
	"github.com/jacopofilonzi/project-library/internal/fsops"
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

	quitting := false
	app := application.New(application.Options{
		Name:        "Project Library",
		Description: "Browse and open the projects in your Development folder",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(lib),
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
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  "main",
		Title: "Project Library",
		Width: 1280, Height: 800,
		// 2 colonne da 220px + scheda progetto da 480px
		MinWidth: 920, MinHeight: 600,
		BackgroundColour: bg,
		URL:              "/",
		// Ctrl/⌘ P non arriva alla pagina (WebView2 la riserva alla stampa): la gestisce la finestra.
		KeyBindings: map[string]func(application.Window){
			"CmdOrCtrl+P": func(application.Window) { app.Event.Emit("shortcut:settings") },
			"CmdOrCtrl+,": func(application.Window) { app.Event.Emit("shortcut:settings") },
		},
	})
	core.MainWindow = win

	// Chiudendo la finestra: con "resta nella tray" si nasconde, altrimenti l'app termina.
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !quitting && store.Get().CloseToTray {
			e.Cancel()
			win.Hide()
			return
		}
		app.Quit()
	})

	lang := func() string { return store.Get().Language }
	menu := application.NewMenu()
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

	// aggiorna le voci della tray quando cambia la lingua
	app.Event.On(core.EventConfig, func(*application.CustomEvent) {
		show.SetLabel(trayLabel(lang(), "show"))
		quit.SetLabel(trayLabel(lang(), "quit"))
		menu.Update()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func trayLabel(lang, key string) string {
	labels := map[string]map[string]string{
		"en": {"show": "Show Project Library", "quit": "Quit"},
		"it": {"show": "Mostra Project Library", "quit": "Esci"},
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

// projectFiles serve le immagini referenziate dai README: /project-file?p=<percorso assoluto>.
// Solo file immagine dentro le cartelle radice.
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
			// gli SVG non devono poter eseguire script
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			http.ServeFile(w, r, p)
		})
	}
}
