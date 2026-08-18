// Package cli は cobra によるコマンド定義を組み立てる。
//
// フラグの値はコマンドごとの options 構造体が持つ。以前はパッケージ変数を
// 全コマンドで共有しており、どのコマンドがどのフラグを持つのか追いにくかった。
package cli

import (
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/analyze"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/analyze/mecab"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/daterange"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/fetch"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/fsutil"
)

// defaultDir は --src / --dest の既定値。
const defaultDir = "~/Desktop"

// options は各コマンドが受け取るフラグの値を保持する。
type options struct {
	src         string
	dest        string
	dates       []string
	maxAttempts int
	timeout     time.Duration
	dicDir      string
	top         int
}

// addSrc は --src フラグを登録する。
func (o *options) addSrc(f *pflag.FlagSet) {
	f.StringVar(&o.src, "src", defaultDir, "source directory path")
}

// addDest は --dest フラグを登録する。
func (o *options) addDest(f *pflag.FlagSet) {
	f.StringVar(&o.dest, "dest", defaultDir, "destination directory path")
}

// addDate は --date フラグを登録する。
func (o *options) addDate(f *pflag.FlagSet) {
	f.StringSliceVar(&o.dates, "date", nil, daterange.FlagUsage)
}

// addHTTP はネットワーク取得に関わるフラグを登録する。
func (o *options) addHTTP(f *pflag.FlagSet) {
	f.IntVar(&o.maxAttempts, "max-attempts", fetch.DefaultMaxAttempts,
		"number of attempts per request (including the first one)")
	f.DurationVar(&o.timeout, "timeout", fetch.DefaultTimeout, "timeout per HTTP request")
}

// addAnalyze は形態素解析に関わるフラグを登録する。
func (o *options) addAnalyze(f *pflag.FlagSet) {
	f.StringVar(&o.dicDir, "dic", "",
		"mecab dictionary directory (default: auto-detect mecab-ipadic-NEologd)")
	f.IntVar(&o.top, "top", analyze.DefaultTop, "number of keywords to keep (0 for all)")
}

// paths は --src / --dest のチルダを展開して返す。
func (o *options) paths() (src, dest string, err error) {
	if src, err = fsutil.ExpandPath(o.src); err != nil {
		return "", "", err
	}
	if dest, err = fsutil.ExpandPath(o.dest); err != nil {
		return "", "", err
	}
	return src, dest, nil
}

// resolveDicDir は --dic の指定値、無ければ自動検出した辞書パスを返す。
func (o *options) resolveDicDir() (string, error) {
	if o.dicDir != "" {
		return fsutil.ExpandPath(o.dicDir)
	}
	return mecab.DefaultDicDir(), nil
}

// newTokenizer は形態素解析器を生成する。呼び出し側が Close する責務を負う。
func (o *options) newTokenizer() (analyze.Tokenizer, error) {
	dicDir, err := o.resolveDicDir()
	if err != nil {
		return nil, err
	}
	slog.Debug("initializing mecab", "dicdir", dicDir)
	return mecab.New(dicDir)
}

// NewRootCommand はルートコマンドを組み立てて返す。
func NewRootCommand() *cobra.Command {
	var verbose bool

	root := &cobra.Command{
		Use:           "wadai",
		Short:         "Rank keywords appearing in Yahoo! News article titles",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(*cobra.Command, []string) {
			setupLogger(verbose)
		},
	}
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable debug logging")

	root.AddCommand(
		newFeedsCommand(),
		newFetchCommand(),
		newExtractCommand(),
		newAnalyzeCommand(),
		newReportCommand(),
		newRunCommand(),
	)
	return root
}

// setupLogger は標準エラーへ出力する構造化ロガーを既定に設定する。
//
// 以前は zap.String(...) が返すフィールド構造体を fmt.Println に渡していたため、
// ログとして読めない出力になっていた。
func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}
