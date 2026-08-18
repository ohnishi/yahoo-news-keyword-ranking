package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/daterange"
)

// newRunCommand はパイプライン全体を一度に実行するコマンドを返す。
func newRunCommand() *cobra.Command {
	o := &options{}
	var forceFetch bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the whole pipeline: fetch feeds, fetch RSS, extract, analyze, report",
		Long: "Run the whole pipeline in one go.\n\n" +
			"Without --date it fetches today's data from the network and processes it.\n" +
			"With --date it only re-processes already fetched data, since RSS cannot be\n" +
			"retrieved for past dates; pass --fetch to force the network stages anyway.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			src, dest, err := o.paths()
			if err != nil {
				return err
			}

			// --date 未指定なら今日を対象に、取得から通す。
			withFetch := forceFetch || len(o.dates) == 0
			dates := o.dates
			if len(dates) == 0 {
				dates = []string{time.Now().Format(daterange.Format)}
			}

			if withFetch {
				ctx := cmd.Context()
				if err := fetchFeeds(ctx, o, dest); err != nil {
					return err
				}
				date, err := singleDateOrToday(dates)
				if err != nil {
					return err
				}
				// 取得したフィード一覧は dest に書かれるので、以降の入力は dest 側になる。
				if err := fetchArticles(ctx, o, dest, dest, date); err != nil {
					return err
				}
				src = dest
			}

			tk, err := o.newTokenizer()
			if err != nil {
				return err
			}
			defer tk.Close()

			return daterange.Each(dates, func(date time.Time) error {
				if err := runExtract(src, dest, date); err != nil {
					return err
				}
				if err := runAnalyze(dest, dest, date, tk, o.top); err != nil {
					return err
				}
				return runReport(dest, dest, date)
			})
		},
	}

	o.addSrc(cmd.Flags())
	o.addDest(cmd.Flags())
	o.addDate(cmd.Flags())
	o.addHTTP(cmd.Flags())
	o.addAnalyze(cmd.Flags())
	cmd.Flags().BoolVar(&forceFetch, "fetch", false,
		"force the network stages even when --date is given")

	return cmd
}
