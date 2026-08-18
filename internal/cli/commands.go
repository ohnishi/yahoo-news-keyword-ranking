package cli

import (
	"context"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/ohnishi/yahoo-news-analysis/internal/analyze"
	"github.com/ohnishi/yahoo-news-analysis/internal/daterange"
	"github.com/ohnishi/yahoo-news-analysis/internal/extract"
	"github.com/ohnishi/yahoo-news-analysis/internal/fetch"
	"github.com/ohnishi/yahoo-news-analysis/internal/report"
)

// newFeedsCommand は RSS フィード一覧を取得するコマンドを返す。
func newFeedsCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "yahoo",
		Short: "Fetch the Yahoo! News RSS feed list into <dest>/rss.jsonl",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, dest, err := o.paths()
			if err != nil {
				return err
			}
			return fetchFeeds(cmd.Context(), o, dest)
		},
	}
	o.addDest(cmd.Flags())
	o.addHTTP(cmd.Flags())
	return cmd
}

// newFetchCommand は各フィードのRSSファイルを取得するコマンドを返す。
func newFetchCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "rss",
		Short: "Fetch RSS files for every feed into <dest>/<YYYYmmdd>/",
		Long: "Fetch RSS files for every feed listed in <src>/rss.jsonl.\n\n" +
			"RSS only ever serves current content, so --date selects the destination\n" +
			"directory rather than the content; it defaults to today.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			src, dest, err := o.paths()
			if err != nil {
				return err
			}
			date, err := singleDateOrToday(o.dates)
			if err != nil {
				return err
			}
			return fetchArticles(cmd.Context(), o, src, dest, date)
		},
	}
	o.addSrc(cmd.Flags())
	o.addDest(cmd.Flags())
	o.addDate(cmd.Flags())
	o.addHTTP(cmd.Flags())
	return cmd
}

// newExtractCommand は取得済みRSSから記事を抽出するコマンドを返す。
func newExtractCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "json",
		Short: "Extract articles published on the target date into <dest>/<YYYYmmdd>/rss.jsonl",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			src, dest, err := o.paths()
			if err != nil {
				return err
			}
			return daterange.Each(o.dates, func(date time.Time) error {
				return runExtract(src, dest, date)
			})
		},
	}
	o.addSrc(cmd.Flags())
	o.addDest(cmd.Flags())
	o.addDate(cmd.Flags())
	_ = cmd.MarkFlagRequired("date")
	return cmd
}

// newAnalyzeCommand は形態素解析でキーワードを抽出するコマンドを返す。
func newAnalyzeCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "analysis",
		Short: "Extract keywords from article titles into <dest>/<YYYYmmdd>/topic.json",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			src, dest, err := o.paths()
			if err != nil {
				return err
			}
			tk, err := o.newTokenizer()
			if err != nil {
				return err
			}
			defer tk.Close()

			return daterange.Each(o.dates, func(date time.Time) error {
				return runAnalyze(src, dest, date, tk, o.top)
			})
		},
	}
	o.addSrc(cmd.Flags())
	o.addDest(cmd.Flags())
	o.addDate(cmd.Flags())
	o.addAnalyze(cmd.Flags())
	_ = cmd.MarkFlagRequired("date")
	return cmd
}

// newReportCommand は Markdown レポートを生成するコマンドを返す。
func newReportCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "markdown",
		Short: "Render the keyword ranking into <dest>/<YYYYmmdd>/report.md",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			src, dest, err := o.paths()
			if err != nil {
				return err
			}
			return daterange.Each(o.dates, func(date time.Time) error {
				return runReport(src, dest, date)
			})
		},
	}
	o.addSrc(cmd.Flags())
	o.addDest(cmd.Flags())
	o.addDate(cmd.Flags())
	_ = cmd.MarkFlagRequired("date")
	return cmd
}

// --- 各段の実行本体。run コマンドからも再利用する。 ---

func fetchFeeds(ctx context.Context, o *options, dest string) error {
	client := fetch.NewClient(fetch.Options{
		Timeout:     o.timeout,
		MaxAttempts: o.maxAttempts,
		Logger:      slog.Default(),
	})
	feeds, err := client.Feeds(ctx)
	if err != nil {
		return err
	}
	if err := fetch.SaveFeeds(dest, feeds); err != nil {
		return err
	}
	slog.Info("fetched rss feed list", "feeds", len(feeds), "dest", dest)
	return nil
}

func fetchArticles(ctx context.Context, o *options, src, dest string, date time.Time) error {
	feeds, err := fetch.LoadFeeds(src)
	if err != nil {
		return err
	}
	client := fetch.NewClient(fetch.Options{
		Timeout:     o.timeout,
		MaxAttempts: o.maxAttempts,
		Logger:      slog.Default(),
	})
	saved, err := client.Articles(ctx, feeds, dest, date)
	if err != nil {
		return err
	}
	slog.Info("fetched rss files", "saved", saved, "feeds", len(feeds),
		"date", date.Format(daterange.Format))
	return nil
}

func runExtract(src, dest string, date time.Time) error {
	n, err := extract.Run(src, dest, date, slog.Default())
	if err != nil {
		return err
	}
	slog.Info("extracted articles", "articles", n, "date", date.Format(daterange.Format))
	return nil
}

func runAnalyze(src, dest string, date time.Time, tk analyze.Tokenizer, top int) error {
	n, err := analyze.Run(src, dest, date, tk, top, slog.Default())
	if err != nil {
		return err
	}
	slog.Info("extracted keywords", "keywords", n, "date", date.Format(daterange.Format))
	return nil
}

func runReport(src, dest string, date time.Time) error {
	if err := report.Run(src, dest, date); err != nil {
		return err
	}
	slog.Info("generated report", "date", date.Format(daterange.Format))
	return nil
}

// singleDateOrToday は --date 未指定なら今日を、1つ指定ならその日を返す。
func singleDateOrToday(values []string) (time.Time, error) {
	if len(values) == 0 {
		return time.Now(), nil
	}
	dates, err := daterange.Parse(values)
	if err != nil {
		return time.Time{}, err
	}
	// 期間を渡されても取得対象は1日なので、先頭を使う。
	return dates[0], nil
}
