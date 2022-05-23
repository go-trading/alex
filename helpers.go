package alex

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//TODO очень не оптимально, надо переделывать, например на деление пополам с кешированием последней найденной позиции
func FindSeries(series *techan.TimeSeries, time time.Time) int {
	if series == nil {
		return -1
	}
	for idx, c := range series.Candles {
		if (time.After(c.Period.Start) && time.Before(c.Period.End)) ||
			time.Equal(c.Period.Start) {
			return idx
		}
	}
	return -1
}

func UpsertSeries(series *techan.TimeSeries, newCandle *techan.Candle) {
	idx := FindSeries(series, newCandle.Period.Start)

	if idx != -1 {
		series.Candles[idx] = newCandle
	} else {
		if !series.AddCandle(newCandle) {
			series.Candles = append(series.Candles, newCandle)
			slices.SortFunc(series.Candles, func(a *techan.Candle, b *techan.Candle) bool {
				return a.Period.Start.Before(b.Period.Start)
			})
		}
	}
}

func getFileName(dataDir string, figi string, period time.Duration) string {
	return path.Join(dataDir, figi+"_"+period.String()+".csv")
}

func LoadTimeSeries(dataDir string, figi string, period time.Duration) (*techan.TimeSeries, error) {
	fileName := getFileName(dataDir, figi, period)
	file, err := os.Open(fileName)
	if err != nil {
		l.Debug("Ранее скаченных файлов со свечами нет", zap.String("fileName", fileName), zap.Error(err))
		return nil, err
	}
	result := techan.NewTimeSeries()
	r := csv.NewReader(bufio.NewReader(file))
	line := 0
	for {
		line++
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			l.Fatal("Ошибка парсинга файла", zap.String("fileName", fileName), zap.Error(err))
		}
		if len(record) != 6 {
			l.Fatal("Количество столбцов отличается от 6", zap.Int("line", line), zap.String("fileName", fileName))
		}
		if line == 1 {
			//пропускаем строку с загоовком
			continue
		}

		t, err := time.Parse("2006-01-02 15:04", record[0])
		if err != nil {
			l.DPanic("time.Parse error",
				zap.String("fileName", fileName),
				zap.Int("line", line),
				zap.Error(err),
			)
		}

		result.AddCandle(&techan.Candle{
			Period:     techan.NewTimePeriod(t, period),
			OpenPrice:  big.NewFromString(record[1]),
			MaxPrice:   big.NewFromString(record[2]),
			MinPrice:   big.NewFromString(record[3]),
			ClosePrice: big.NewFromString(record[4]),
			Volume:     big.NewFromString(record[5]),
		})
	}
	return result, nil
}

func SaveTimeSeries(dataDir string, figi string, period time.Duration, timeSeries *techan.TimeSeries) error {
	fileName := getFileName(dataDir, figi, period)
	path := filepath.Dir(fileName)
	if err := os.MkdirAll(path, os.ModePerm); err != nil && !os.IsExist(err) {
		l.DPanic("не смог создать каталог",
			zap.String("path", path),
			zap.Error(err))
		return err
	}

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		l.DPanic("не открыть файл",
			zap.String("fileName", fileName),
			zap.Error(err))
		return err
	}
	defer file.Close()

	datawriter := bufio.NewWriter(file)
	defer datawriter.Flush()

	datawriter.WriteString("Time,Open,High,Low,Close,Volume\n") //nolint:golint,errcheck
	for _, candle := range timeSeries.Candles {
		_, err = datawriter.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			candle.Period.Start.Format("2006-01-02 15:04"),
			candle.OpenPrice,
			candle.MaxPrice,
			candle.MinPrice,
			candle.ClosePrice,
			candle.Volume,
		))
		if err != nil {
			l.DPanic("не смог записать в файл",
				zap.String("fileName", fileName),
				zap.Error(err))
			return err
		}
	}
	return nil
}

//TODO надо вводить свой класс ошибок
func IsLimitError(e error) bool {
	if se, ok := e.(interface {
		GRPCStatus() *status.Status
	}); ok {
		l.Error("se.GRPCStatus()",
			zap.Any("GRPCStatus", se.GRPCStatus()),
			zap.String("Code", se.GRPCStatus().Code().String()),
			zap.Any("Message", se.GRPCStatus().Message()),
		)
		return se.GRPCStatus().Code() == codes.InvalidArgument &&
			se.GRPCStatus().Message() == "30042"
	}
	return false
}
