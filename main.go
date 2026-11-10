package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/feiyushaw/ztemate/internal/app"
	"github.com/feiyushaw/ztemate/internal/store"
)

func main() {
	dir, e := os.UserConfigDir()
	if e != nil {
		log.Fatal(e)
	}
	db, e := store.Open(filepath.Join(dir, "ZTEMate", "ztemate.db"))
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	var mainWindow, panel *mygo.Window
	var tray *mygo.Tray
	quitting := false
	openMain := func() {
		mygo.RunOnMain(func() {
			if panel != nil {
				panel.Close()
			}
			if mainWindow != nil {
				mainWindow.Show()
				mainWindow.Focus()
			}
		})
	}
	service, e := app.New(db, app.Hooks{Copy: mygo.Clipboard.WriteText, OpenMain: openMain, OpenAdmin: mygo.Shell.OpenExternal, SetLogin: mygo.App.SetOpenAtLogin, Notify: func(body string) {
		n := mygo.NewNotification(mygo.NotificationOptions{Title: "ZTEMate · 新短信", Body: body})
		n.OnClick(openMain)
		if err := n.Show(); err != nil {
			log.Printf("notification: %v", err)
		}
	}})
	if e != nil {
		log.Fatal(e)
	}
	mygo.Bind(service)
	mygo.App.SetName("ZTEMate")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true; service.Stop() })
	mygo.App.WhenReady(func() {
		mainWindow = mygo.NewWindow(mygo.WindowOptions{Title: "ZTEMate", Width: 1040, Height: 740, MinWidth: 780, MinHeight: 580, URL: "/", StateKey: "main", BackgroundColor: "light-dark(#f6f7f9, #171a20)", Hidden: true})
		mainWindow.OnReadyToShow(mainWindow.Show)
		mainWindow.OnFocus(func() { service.SetActive(true) })
		mainWindow.OnBlur(func() { service.SetActive(false) })
		mainWindow.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				mainWindow.Hide()
				service.SetActive(false)
			}
		})
		var err error
		tray, err = mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: true, ToolTip: "ZTEMate · 网络与短信"})
		if err != nil {
			log.Print(err)
			return
		}
		tray.OnClick(func() {
			if panel != nil {
				panel.Close()
				return
			}
			panel = mygo.NewFlyout(mygo.FlyoutOptions{Tray: tray, Placement: mygo.PlacementBottom, Width: 360, Height: 520, Focusable: true, Shadow: true, Popover: true, URL: "/?view=tray"})
			service.SetActive(true)
			panel.OnClosed(func() { panel = nil; service.SetActive(false) })
		})
		menu := mygo.NewMenu([]*mygo.MenuItem{{Label: "打开 ZTEMate", Click: func(*mygo.MenuItem, *mygo.Window) { openMain() }}, {Label: "暂停监控", Click: func(*mygo.MenuItem, *mygo.Window) { go service.Disconnect() }}, mygo.Separator(), {Role: mygo.RoleQuit}})
		tray.OnRightClick(func() { tray.SetMenu(menu); defer tray.SetMenu(nil); tray.PopUpMenu() })
		mygo.Power.OnSuspend(func() { service.SetSleeping(true) })
		mygo.Power.OnResume(func() { service.SetSleeping(false) })
		service.Start()
	})
	if e = mygo.App.Run(); e != nil {
		log.Fatal(e)
	}
}
func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 36, 36))
	for i := 0; i < 4; i++ {
		h := 6 + i*5
		for x := 4 + i*8; x < 9+i*8; x++ {
			for y := 30 - h; y < 30; y++ {
				img.Set(x, y, color.NRGBA{A: 255})
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
