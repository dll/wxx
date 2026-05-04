package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func escapeQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return "\"\""
	}

	q = strings.ReplaceAll(q, "\"", "")

	runes := []rune(q)
	hasChinese := false
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			hasChinese = true
			break
		}
	}

	if hasChinese {
		var parts []string
		for _, r := range runes {
			if r >= 0x4E00 && r <= 0x9FFF {
				parts = append(parts, string(r)+"*")
			}
		}
		if len(parts) == 0 {
			return "\"\""
		}
		return strings.Join(parts, " OR ")
	}

	return "\"" + q + "\""
}

func main() {
	db, err := sql.Open("sqlite3", "./data/wxx.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := "国家奖学金"
	escaped := escapeQuery(query)
	fmt.Printf("原始查询: %s\n", query)
	fmt.Printf("转义后: %s\n\n", escaped)

	rows, err := db.Query(`
		SELECT kb.title, bm25(kb_fts) AS score
		FROM kb_fts
		JOIN kb_resources kb ON kb_fts.rowid = kb.id
		WHERE kb_fts MATCH ?
		  AND kb.status = 'published'
		  AND kb.owner_scope = 'school'
		  AND kb.role_scope LIKE '%student%'
		ORDER BY score
		LIMIT 3
	`, escaped)
	if err != nil {
		log.Fatal("查询失败:", err)
	}
	defer rows.Close()

	fmt.Println("查询结果:")
	count := 0
	for rows.Next() {
		var title string
		var score float64
		rows.Scan(&title, &score)
		fmt.Printf("  - %s (score: %.2f)\n", title, score)
		count++
	}
	if count == 0 {
		fmt.Println("  (无结果)")
	}
}
