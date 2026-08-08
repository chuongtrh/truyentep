package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"truyentep/internal/truyentep"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Truyền Tệp:", err)
		(truyentep.SystemNotifier{}).Notify("Truyền Tệp không thể khởi động", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := truyentep.DefaultConfig()
	if err != nil {
		return err
	}
	showVersion := false
	flag.StringVar(&cfg.Name, "name", cfg.Name, "tên máy hiển thị")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "cổng truyền dữ liệu")
	flag.IntVar(&cfg.DiscoveryPort, "discovery-port", cfg.DiscoveryPort, "cổng tìm máy trong mạng nội bộ")
	flag.StringVar(&cfg.DownloadDir, "downloads", cfg.DownloadDir, "thư mục lưu tệp nhận")
	flag.BoolVar(&cfg.NoOpen, "no-open", false, "không tự mở trình duyệt")
	flag.BoolVar(&cfg.NoDiscovery, "no-discovery", false, "tắt tìm máy tự động")
	flag.BoolVar(&showVersion, "version", false, "in phiên bản")
	flag.Parse()

	if showVersion {
		fmt.Println("Truyền Tệp", truyentep.Version)
		return nil
	}

	app, err := truyentep.New(cfg)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = app.Run(ctx)
	if errors.Is(err, truyentep.ErrAlreadyRunning) {
		return truyentep.OpenBrowser(app.URL())
	}
	return err
}
