package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/prismbox/prismbox/internal/config"
	"github.com/prismbox/prismbox/internal/kernel"
	"github.com/prismbox/prismbox/internal/monitor"
	"github.com/prismbox/prismbox/internal/server"
)

// Build metadata injected via ldflags
var (
	Version   = "1.0.0"
	BuildTime = ""
	Commit    = ""
)

func main() {
	listenAddr := flag.String("listen", ":9090", "Address and port to listen on")
	dataDirFlag := flag.String("data-dir", "~/.prismbox", "Directory for PrismBox configurations and runtime data")
	kernelPathFlag := flag.String("kernel", "", "Path to sing-box binary (defaults to auto-detect in ./sing-box or $PATH)")
	flag.Parse()

	// 1. Resolve user home directory expansion if "~" or "~/" is used
	dataDir := *dataDirFlag
	if strings.HasPrefix(dataDir, "~/") || dataDir == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			if dataDir == "~" {
				dataDir = home
			} else {
				dataDir = filepath.Join(home, dataDir[2:])
			}
		}
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory '%s': %v", dataDir, err)
	}

	// 2. Auto-detect sing-box binary
	kernelPath, err := kernel.FindKernelBinary(*kernelPathFlag)
	if err != nil {
		if *kernelPathFlag != "" {
			log.Printf("[kernel] Warning: specified kernel binary not found: %v", err)
			kernelPath = *kernelPathFlag
		} else {
			log.Printf("[kernel] Warning: sing-box binary not found in ./sing-box or PATH. Kernel will remain offline.")
			kernelPath = "sing-box"
		}
	}

	// 3. Initialize config.Store
	store, err := config.NewStore(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize config store: %v", err)
	}

	// 4. Initialize kernel.Manager
	mgr := kernel.NewManager(kernelPath, dataDir)
	mgr.SetStore(store)

	// 5. Initialize monitor.Monitor
	mon := monitor.NewMonitor()

	// 6. Initialize server.Server
	srv := server.New(*listenAddr, store, mgr, mon)

	// Print startup banner
	printBanner(*listenAddr, dataDir, kernelPath)

	// Start HTTP server in a separate goroutine
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start kernel automatically on boot
	if err := mgr.Start(); err != nil {
		log.Printf("[kernel] Notice: could not automatically start sing-box on boot: %v", err)
	}

	// Graceful shutdown handling for SIGINT and SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal %s, initiating graceful shutdown...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	mon.Stop()
	if err := mgr.Stop(); err != nil {
		log.Printf("Error stopping kernel process: %v", err)
	}

	log.Println("PrismBox exited cleanly.")
}

func printBanner(listenAddr, dataDir, kernelPath string) {
	ver := Version
	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}
	banner := `
   ____       _               ____             
  |  _ \ _ __(_)___ _ __ ___ | __ )  _____  __ 
  | |_) | '__| / __| '_ ` + "`" + ` _ \|  _ \ / _ \ \/ / 
  |  __/| |  | \__ \ | | | | | |_) | (_) >  <  
  |_|   |_|  |_|___/_| |_| |_|____/ \___/_/\_\ 
  PrismBox - WebUI Manager for sing-box ` + ver + `
`
	fmt.Print(banner)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf(" Listening WebUI & API: http://%s\n", formatListenAddr(listenAddr))
	fmt.Printf(" Data directory       : %s\n", dataDir)
	fmt.Printf(" sing-box binary      : %s\n", kernelPath)
	fmt.Println(strings.Repeat("-", 60))
}

func formatListenAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	return addr
}
