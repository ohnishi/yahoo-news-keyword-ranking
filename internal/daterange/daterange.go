// Package daterange は `--date` フラグで指定された日付・期間の解釈を担う。
package daterange

import (
	"errors"
	"fmt"
	"time"
)

// Format は `--date` フラグおよび日付ディレクトリ名で用いるレイアウト。
const Format = "20060102"

// FlagUsage は `--date` フラグのヘルプ文言。
const FlagUsage = "target date in 'YYYYmmdd', or period in 'YYYYmmdd,YYYYmmdd' " +
	"(e.g. '20201218' or '20201201,20201231')"

// Parse は `--date` の値を対象日の一覧へ展開する。
//
// 値が1つなら単日、2つなら両端を含む期間を1日刻みで展開する。
// 期間が逆順に指定された場合は入れ替えて解釈する。
func Parse(values []string) ([]time.Time, error) {
	switch len(values) {
	case 0:
		return nil, errors.New("date must be specified as 'YYYYmmdd' or 'YYYYmmdd,YYYYmmdd'")
	case 1:
		d, err := ParseDate(values[0])
		if err != nil {
			return nil, err
		}
		return []time.Time{d}, nil
	case 2:
		since, err := ParseDate(values[0])
		if err != nil {
			return nil, err
		}
		until, err := ParseDate(values[1])
		if err != nil {
			return nil, err
		}
		if since.After(until) {
			since, until = until, since
		}
		var dates []time.Time
		for d := since; !d.After(until); d = d.AddDate(0, 0, 1) {
			dates = append(dates, d)
		}
		return dates, nil
	default:
		return nil, fmt.Errorf("date accepts at most 2 values, but got %d", len(values))
	}
}

// ParseDate は "YYYYmmdd" をローカルタイムゾーンの日付として解釈する。
func ParseDate(value string) (time.Time, error) {
	d, err := time.ParseInLocation(Format, value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %q as %q: %w", value, Format, err)
	}
	return d, nil
}

// Each は各対象日に対して fn を呼ぶ。
// ある日が失敗しても残りの日は処理し、最後に全エラーをまとめて返す。
func Each(values []string, fn func(time.Time) error) error {
	dates, err := Parse(values)
	if err != nil {
		return err
	}
	var errs []error
	for _, d := range dates {
		if err := fn(d); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.Format(Format), err))
		}
	}
	return errors.Join(errs...)
}
