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
		fmt.Println("no window opened, leave this running and open", url)
		select {}
	}
	fmt.Println("window", url)
	_ = cmd.Wait()
	time.Sleep(200 * time.Millisecond)
}

func openWindow(url string) *exec.Cmd {
	if runtime.GOOS != "windows" {
		cmd := exec.Command("xdg-open", url)
		_ = cmd.Start()
		return nil
	}
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LocalAppData"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
	}
	for _, bin := range candidates {
		if _, err := os.Stat(bin); err != nil {
			continue
		}
		cmd := exec.Command(bin, "--app="+url, "--window-size=980,720")
		if err := cmd.Start(); err == nil {
			return cmd
		}
	}
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err == nil {
		return nil
	}
	return nil
}
