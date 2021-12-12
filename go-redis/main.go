package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/hhxsv5/go-redis-memory-analysis"
)

var ctx context.Context
var rdb redis.UniversalClient

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     128,
		MinIdleConns: 100,
		MaxRetries:   5,
	})
	ctx = context.Background()
}

func main() {
	batchSet(10000, "size10_10k", generateValue(10))
	batchSet(100000, "size10_100k", generateValue(10))
	batchSet(500000, "size10_500k", generateValue(10))

	batchSet(10000, "size100_10k", generateValue(100))
	batchSet(100000, "size100_100k", generateValue(100))
	batchSet(500000, "size100_500k", generateValue(100))

	batchSet(10000, "size1000_10k", generateValue(1000))
	batchSet(100000, "size1000_100k", generateValue(1000))
	batchSet(500000, "size1000_500k", generateValue(1000))

	batchSet(10000, "size5000_10k", generateValue(5000))
	batchSet(100000, "size5000_100k", generateValue(5000))
	batchSet(500000, "size5000_500k", generateValue(5000))

	analysisMemory()
}

func batchSet(num int, keyPrefix, value string) {
	for i := 0; i < num; i++ {
		key := fmt.Sprintf("%s:%v", keyPrefix, i)
		set := rdb.Set(ctx, key, value, -1)
		err := set.Err()
		if err != nil {
			fmt.Println(set.String())
		}
	}
}

func generateValue(size int) string {
	bytes := make([]byte, size)
	for i := 0; i < size; i++ {
		bytes[i] = byte(i)
	}
	return string(bytes)
}

func analysisMemory() {
	analysis, err := gorma.NewAnalysisConnection("localhost", 6379, "")
	if err != nil {
		fmt.Println("analysis created failed:", err)
		return
	}

	analysis.Start([]string{":"})

	err = analysis.SaveReports("./reports")
	if err == nil {
		fmt.Println("analysis done")
	} else {
		fmt.Println("analysis failed:", err)
	}
}
