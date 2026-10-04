package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/5mil/arcis-windows/internal/host"
)

func main() {
	port := flag.Int("port", 9090, "local port")
	tier := flag.String("tier", "visio", "forma|figura|visio")
	self := flag.Bool("self-test", false, "run the contained host self-test and exit")
	noWindow := flag.Bool("no-window", false, "serve only, do not open the desktop window")
	flag.Parse()

	if *self {
		os.Exit(host.SelfTest())
	}

	data := host.DataDir()
	h := host.New(data, *tier)
	addr, err := h.Listen(*port)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("arcis.exe listening on http://%s  data=%s\n", addr, data)
	// Server is up. Light weights were generated if missing.
	// Free house GGUFs influence the design, so pull them without waiting on a click.
	go h.StartHousePull()
	if *noWindow {
		select {}
	}
	url := "http://" + addr + "/"
	cmd := openWindow(url)
	if cmd == nil {
		fmt.Println("open", url)
		select {}
	}
	_ = cmd.Wait()
	// Edge/Chrome app mode exits when the window closes.
	time.Sleep(200 * time.Millisecond)
}

func openWindow(url string) *exec.Cmd {
	if runtime.GOOS != "windows" {
		cmd := exec.Command("xdg-open", url)
		_ = cmd.Start()
		return nil
	}
	edge := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe")
	if _, err := os.Stat(edge); err != nil {
		edge = filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe")
	}
	if _, err := os.Stat(edge); err == nil {
		return exec.Command(edge, "--app="+url, "--window-size=980,680")
	}
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	_ = cmd.Start()
	return nil
}
