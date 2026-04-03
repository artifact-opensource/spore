package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/artifact-virtual/symbiote-android/core"
)

const (
	chunkSize    = 512
	chunkOverlap = 64
)

type Result = core.SearchResult

type Document struct {
	Path   string   `json:"path"`
	Chunks []string `json:"chunks"`
}

type Store struct {
	dataDir string
	docs    []Document
	bm25    *BM25Index
}

func New(dataDir string) *Store {
	s := &Store{
		dataDir: dataDir,
		bm25:    NewBM25Index(),
	}
	s.load()
	return s
}

func (s *Store) Search(query string, k int) []Result {
	if len(s.docs) == 0 {
		return nil
	}
	results := s.bm25.Search(query, k*2)
	if len(results) > k {
		results = results[:k]
	}
	return results
}

func (s *Store) Ingest(path string) (int, error) {
	if !filepath.IsAbs(path) {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path)
	}

	count := 0
	s.docs = nil
	s.bm25 = NewBM25Index()

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			// Skip Android system dirs
			base := info.Name()
			if base == "node_modules" || base == "__pycache__" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		if info.Size() > 1<<20 {
			return nil
		}
		if !isTextFile(info.Name()) {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}

		content := string(data)
		chunks := chunkText(content, chunkSize, chunkOverlap)

		relPath, _ := filepath.Rel(path, p)
		if relPath == "" {
			relPath = p
		}

		doc := Document{Path: relPath, Chunks: chunks}
		s.docs = append(s.docs, doc)

		for _, chunk := range chunks {
			s.bm25.Add(relPath, chunk)
		}

		count++
		return nil
	})

	if err != nil {
		return count, err
	}

	s.save()
	return count, nil
}

func (s *Store) Stats() core.MemoryStats {
	totalChunks := 0
	for _, d := range s.docs {
		totalChunks += len(d.Chunks)
	}

	indexSize := int64(0)
	indexPath := filepath.Join(s.dataDir, "memory", "index.json")
	if info, err := os.Stat(indexPath); err == nil {
		indexSize = info.Size()
	}

	return core.MemoryStats{
		Documents:  len(s.docs),
		Vectors:    0,
		IndexBytes: indexSize,
	}
}

func (s *Store) save() {
	dir := filepath.Join(s.dataDir, "memory")
	os.MkdirAll(dir, 0755)
	data, _ := json.Marshal(s.docs)
	os.WriteFile(filepath.Join(dir, "index.json"), data, 0644)
}

func (s *Store) load() {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "memory", "index.json"))
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.docs)
	for _, doc := range s.docs {
		for _, chunk := range doc.Chunks {
			s.bm25.Add(doc.Path, chunk)
		}
	}
}

// --- BM25 ---

type BM25Index struct {
	entries []bm25Entry
	df      map[string]int
	avgDL   float64
}

type bm25Entry struct {
	path  string
	chunk string
	terms map[string]int
	len   int
}

func NewBM25Index() *BM25Index {
	return &BM25Index{df: make(map[string]int)}
}

func (idx *BM25Index) Add(path, chunk string) {
	terms := tokenize(chunk)
	tf := make(map[string]int)
	for _, t := range terms {
		tf[t]++
	}

	seen := make(map[string]bool)
	for _, t := range terms {
		if !seen[t] {
			idx.df[t]++
			seen[t] = true
		}
	}

	idx.entries = append(idx.entries, bm25Entry{
		path: path, chunk: chunk, terms: tf, len: len(terms),
	})

	total := 0
	for _, e := range idx.entries {
		total += e.len
	}
	idx.avgDL = float64(total) / float64(len(idx.entries))
}

func (idx *BM25Index) Search(query string, k int) []Result {
	queryTerms := tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	n := float64(len(idx.entries))
	k1 := 1.5
	b := 0.75

	type scored struct {
		idx   int
		score float64
	}

	var scores []scored
	for i, entry := range idx.entries {
		score := 0.0
		for _, qt := range queryTerms {
			tf := float64(entry.terms[qt])
			if tf == 0 {
				continue
			}
			df := float64(idx.df[qt])
			idf := math.Log((n-df+0.5)/(df+0.5) + 1)
			tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*float64(entry.len)/idx.avgDL))
			score += idf * tfNorm
		}
		if score > 0 {
			scores = append(scores, scored{i, score})
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if len(scores) > k {
		scores = scores[:k]
	}

	results := make([]Result, len(scores))
	for i, s := range scores {
		results[i] = Result{
			Path:  idx.entries[s.idx].path,
			Score: s.score,
			Chunk: idx.entries[s.idx].chunk,
		}
	}

	return results
}

// --- Utils ---

func tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tok := current.String()
				if len(tok) > 1 && !isStopWord(tok) {
					tokens = append(tokens, tok)
				}
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tok := current.String()
		if len(tok) > 1 && !isStopWord(tok) {
			tokens = append(tokens, tok)
		}
	}

	return tokens
}

func chunkText(text string, size, overlap int) []string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	var chunks []string
	var current strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if current.Len()+len(line)+1 > size && current.Len() > 0 {
			chunks = append(chunks, current.String())
			content := current.String()
			current.Reset()
			if len(content) > overlap {
				current.WriteString(content[len(content)-overlap:])
			}
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

func isTextFile(name string) bool {
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true,
		".js": true, ".ts": true, ".json": true, ".yaml": true,
		".yml": true, ".toml": true, ".cfg": true, ".conf": true,
		".sh": true, ".bash": true, ".c": true, ".h": true,
		".cpp": true, ".rs": true, ".java": true, ".rb": true,
		".html": true, ".css": true, ".xml": true, ".csv": true,
		".log": true, ".env": true, ".ini": true, ".sql": true,
		".kt": true, ".swift": true, ".lua": true, ".r": true,
	}
	ext := strings.ToLower(filepath.Ext(name))
	return textExts[ext]
}

func isStopWord(w string) bool {
	stops := map[string]bool{
		"the": true, "is": true, "at": true, "of": true,
		"on": true, "in": true, "to": true, "and": true,
		"or": true, "an": true, "it": true, "as": true,
		"be": true, "by": true, "do": true, "for": true,
		"from": true, "has": true, "he": true, "if": true,
		"no": true, "not": true, "she": true, "so": true,
		"that": true, "this": true, "was": true, "we": true,
		"with": true, "you": true, "are": true, "but": true,
		"had": true, "have": true, "his": true, "her": true,
		"its": true, "may": true, "my": true, "our": true,
		"than": true, "them": true, "they": true, "too": true,
		"very": true, "what": true, "when": true, "who": true,
		"will": true, "your": true, "a": true,
	}
	return stops[w]
}

func Fmt(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}
