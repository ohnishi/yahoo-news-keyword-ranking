// Command yahoo-news-analysis は Yahoo!ニュースの記事タイトルから
// 形態素解析でキーワードを抽出し、ランキングのMarkdownレポートを生成する。
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ohnishi/yahoo-news-analysis/internal/cli"
)

func main() {
	// Ctrl-C で進行中のHTTP取得を中断できるようにする。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.NewRootCommand().ExecuteContext(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
