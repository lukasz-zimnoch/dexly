package inmem

import (
	"github.com/lukasz-zimnoch/dexly/trading"
	"testing"
	"time"
)

func TestCandleRepository_SaveCandles(t *testing.T) {
	windowSize := 5
	repository := NewCandleRepository(windowSize)

	candles := []*trading.Candle{
		candle(t, "2021-06-11T15:00:00Z", "2021-06-11T15:00:59Z"),
		candle(t, "2021-06-11T15:00:00Z", "2021-06-11T15:00:59Z"),
		candle(t, "2021-06-11T15:01:00Z", "2021-06-11T15:01:59Z"),
		candle(t, "2021-06-11T15:02:00Z", "2021-06-11T15:02:59Z"),
		candle(t, "2021-06-11T15:03:00Z", "2021-06-11T15:03:59Z"),
		candle(t, "2021-06-11T15:04:00Z", "2021-06-11T15:04:59Z"),
		candle(t, "2021-06-11T15:04:00Z", "2021-06-11T15:04:59Z"),
		candle(t, "2021-06-11T15:05:00Z", "2021-06-11T15:05:59Z"),
		candle(t, "2021-06-11T15:06:00Z", "2021-06-11T15:06:59Z"),
		candle(t, "2021-06-11T15:07:00Z", "2021-06-11T15:07:59Z"),
	}

	repository.SaveCandles("key", candles...)

	actualCandles := repository.Candles("key")

	if len(actualCandles) != windowSize {
		t.Errorf(
			"unexpected candles count\n"+
				"expected: [%v]\n"+
				"actual:   [%v]",
			windowSize,
			len(actualCandles),
		)
	}

	assertCandlesEqual(
		t,
		candle(t, "2021-06-11T15:03:00Z", "2021-06-11T15:03:59Z"),
		actualCandles[0],
	)
	assertCandlesEqual(
		t,
		candle(t, "2021-06-11T15:04:00Z", "2021-06-11T15:04:59Z"),
		actualCandles[1],
	)
	assertCandlesEqual(
		t,
		candle(t, "2021-06-11T15:05:00Z", "2021-06-11T15:05:59Z"),
		actualCandles[2],
	)
	assertCandlesEqual(
		t,
		candle(t, "2021-06-11T15:06:00Z", "2021-06-11T15:06:59Z"),
		actualCandles[3],
	)
	assertCandlesEqual(
		t,
		candle(t, "2021-06-11T15:07:00Z", "2021-06-11T15:07:59Z"),
		actualCandles[4],
	)
}

func TestCandleRepository_DeleteCandles(t *testing.T) {
	windowSize := 5
	repository := NewCandleRepository(windowSize)

	candles := []*trading.Candle{
		candle(t, "2021-06-11T15:00:00Z", "2021-06-11T15:00:59Z"),
		candle(t, "2021-06-11T15:01:00Z", "2021-06-11T15:01:59Z"),
	}

	repository.SaveCandles("key", candles...)

	repository.DeleteCandles("key")

	expectedCandlesCount := 0
	actualCandlesCount := len(repository.Candles("key"))

	if actualCandlesCount != expectedCandlesCount {
		t.Errorf(
			"unexpected candles count\n"+
				"expected: [%v]\n"+
				"actual:   [%v]",
			expectedCandlesCount,
			actualCandlesCount,
		)
	}
}

func assertCandlesEqual(
	t *testing.T,
	expected *trading.Candle,
	actual *trading.Candle,
) {
	if !expected.Equal(actual) {
		t.Errorf(
			"unexpected candle\n"+
				"expected: [%v]\n"+
				"actual:   [%v]",
			expected.String(),
			actual.String(),
		)
	}
}

func candle(t *testing.T, openTime, closeTime string) *trading.Candle {
	return &trading.Candle{
		OpenTime:  parseTime(t, openTime),
		CloseTime: parseTime(t, closeTime),
	}
}

func parseTime(t *testing.T, value string) time.Time {
	time, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}

	return time
}
