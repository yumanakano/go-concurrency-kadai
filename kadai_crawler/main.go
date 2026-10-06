package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// 訪問済みURLを管理
type SafeUrlMap struct {
	visited map[string]bool
	mu      sync.Mutex
}

// URLが未訪問なら登録してtrueを返す
func (s *SafeUrlMap) Visit(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.visited[url] {
		return false
	}

	s.visited[url] = true
	return true
}

// URLを受け取り、そのページに含まれるリンクを返す
func fetch(url string) []string {
	// 0〜1.3秒の通信遅延をシミュレーション
	time.Sleep(time.Duration(rand.Intn(1300)) * time.Millisecond)

	links := map[string][]string{
		"https://example.com": {
			"https://example.com/a",
			"https://example.com/b",
		},
		"https://example.com/a": {
			"https://example.com/c",
		},
		"https://example.com/b": {
			"https://example.com/a", // 重複リンク
		},
		"https://example.com/c": {},
	}

	return links[url]
}

// 再帰的にクロールする
func Crawl(url string, visited *SafeUrlMap, wg *sync.WaitGroup) {
	defer wg.Done()

	// 訪問済みなら終了
	if !visited.Visit(url) {
		return
	}

	fmt.Println("Crawling:", url)

	// fetch結果を受け取るバッファありチャネル
	resultCh := make(chan []string, 1)

	go func() {
		resultCh <- fetch(url)
	}()

	var links []string

	// fetch完了 or タイムアウトを待つ
	select {
	case links = <-resultCh:
		fmt.Println("Success:", url)

	case <-time.After(1 * time.Second):
		fmt.Println("Timeout:", url)
		return
	}

	// リンク先を再帰的にクロール
	for _, link := range links {
		wg.Add(1)
		go Crawl(link, visited, wg)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	visited := &SafeUrlMap{
		visited: make(map[string]bool),
	}

	var wg sync.WaitGroup

	// 最初のURLから開始
	wg.Add(1)
	go Crawl("https://example.com", visited, &wg)

	// 全goroutineの終了を待機
	wg.Wait()

	fmt.Println("Finished")
}
