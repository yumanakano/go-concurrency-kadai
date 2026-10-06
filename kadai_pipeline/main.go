package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// テスト用のダミーファイルを作成する関数
func generateDummyFiles(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 5つのファイルを作成
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))

		// "word " を i*10 個作成
		content := strings.Repeat("word ", i*10)

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// ステージ1: ファイル探索
func findFiles(ctx context.Context, root string, paths chan<- string) {
	defer close(paths)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// ディレクトリはスキップ
		if d.IsDir() {
			return nil
		}

		// .txtのみ対象
		if filepath.Ext(path) == ".txt" {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case paths <- path:
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("探索エラー: %v\n", err)
	}
}

// ステージ2: 単語数集計
func countWords(ctx context.Context, paths <-chan string, counts chan<- int) {
	for {
		select {
		case <-ctx.Done():
			return

		case path, ok := <-paths:
			if !ok {
				return
			}

			file, err := os.Open(path)
			if err != nil {
				fmt.Printf("ファイルオープンエラー: %v\n", err)
				continue
			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanWords)

			count := 0
			for scanner.Scan() {
				count++
			}

			file.Close()

			if err := scanner.Err(); err != nil {
				fmt.Printf("読み込みエラー: %v\n", err)
				continue
			}

			select {
			case <-ctx.Done():
				return

			case counts <- count:
			}
		}
	}
}

func main() {
	rootDir := "tmp"

	// ダミーファイル生成
	if err := generateDummyFiles(rootDir); err != nil {
		fmt.Printf("ファイル生成エラー: %v\n", err)
		return
	}

	// 30秒タイムアウト
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	// キャンセル動作確認する場合はこちら
	// ctx, cancel := context.WithTimeout(
	//     context.Background(),
	//     1*time.Millisecond,
	// )
	// defer cancel()

	paths := make(chan string)
	counts := make(chan int)

	// ステージ1
	go findFiles(ctx, rootDir, paths)

	// ステージ2 (Worker Pool)
	var wg sync.WaitGroup

	workerCount := 4

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			countWords(ctx, paths, counts)
		}()
	}

	// Worker終了後にcountsを閉じる
	go func() {
		wg.Wait()
		close(counts)
	}()

	// ステージ3: 集計
	total := 0

	for count := range counts {
		total += count
	}

	fmt.Printf("総単語数: %d\n", total)
}
