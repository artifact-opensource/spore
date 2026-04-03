package memory

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/artifact-virtual/spore/core"
)

const (
	chunkSize    = 512 // chars per chunk
	chunkOverlap = 64
	vectorDim    = 0 // set when first embedding arrives; 0 = BM25 only
)

// Result from a search — implements core.SearchResult
type Result = core.SearchResult

// Document stored in the index
type Document struct {
	Path   string   `json:"path"`
	Chunks []string `json:"chunks"`
}

// Store is the memory system: BM25 + optional vector
type Store struct {
	workspace string
	docs      []Document
	bm25      *BM25Index
	vectors   *VectorIndex
}

func New(workspace string) *Store {
	s := &Store{
		workspace: workspace,
		bm25:      NewBM25Index(),
		vectors:   nil,
	}
	s.load()
	return s
}

func (s *Store) Search(query string, k int) []Result {
	if len(s.docs) == 0 {
		return nil
	}

	results := s.bm25.Search(query, k*2) // get extra for dedup

	// if vectors available, do hybrid
	if s.vectors != nil && s.vectors.Count() > 0 {
		// for now BM25 only — vector search requires embedding endpoint
		// which we'll add when llamafile is running
	}

	// cap to k
	if len(results) > k {
		results = results[:k]
	}

	return results
}

func (s *Store) Ingest(path string) (int, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.workspace, path)
	}

	count := 0
	s.docs = nil
	s.bm25 = NewBM25Index()

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			// skip hidden dirs
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		// skip binary, large, hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		if info.Size() > 1<<20 { // skip >1MB
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

		doc := Document{
			Path:   relPath,
			Chunks: chunks,
		}
		s.docs = append(s.docs, doc)

		// index in BM25
		for _, chunk := range chunks {
			s.bm25.Add(relPath, chunk)
		}

		count++
		return nil
	})

	if err != nil {
		return count, err
	}

	// save index
	s.save()

	return count, nil
}

func (s *Store) Stats() core.MemoryStats {
	totalChunks := 0
	for _, d := range s.docs {
		totalChunks += len(d.Chunks)
	}

	indexSize := int64(0)
	indexPath := filepath.Join(s.workspace, "index.json")
	if info, err := os.Stat(indexPath); err == nil {
		indexSize = info.Size()
	}

	vectorCount := 0
	if s.vectors != nil {
		vectorCount = s.vectors.Count()
	}

	return core.MemoryStats{
		Documents:  len(s.docs),
		Vectors:    vectorCount,
		IndexBytes: indexSize,
	}
}

func (s *Store) save() {
	os.MkdirAll(s.workspace, 0755)
	data, _ := json.Marshal(s.docs)
	os.WriteFile(filepath.Join(s.workspace, "index.json"), data, 0644)
}

func (s *Store) load() {
	data, err := os.ReadFile(filepath.Join(s.workspace, "index.json"))
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.docs)

	// rebuild BM25 from loaded docs
	for _, doc := range s.docs {
		for _, chunk := range doc.Chunks {
			s.bm25.Add(doc.Path, chunk)
		}
	}
}

// --- BM25 Index ---

type BM25Index struct {
	entries []bm25Entry
	df      map[string]int // document frequency
	avgDL   float64
}

type bm25Entry struct {
	path  string
	chunk string
	terms map[string]int
	len   int
}

func NewBM25Index() *BM25Index {
	return &BM25Index{
		df: make(map[string]int),
	}
}

func (idx *BM25Index) Add(path, chunk string) {
	terms := tokenize(chunk)
	tf := make(map[string]int)
	for _, t := range terms {
		tf[t]++
	}

	// update doc frequency
	seen := make(map[string]bool)
	for _, t := range terms {
		if !seen[t] {
			idx.df[t]++
			seen[t] = true
		}
	}

	idx.entries = append(idx.entries, bm25Entry{
		path:  path,
		chunk: chunk,
		terms: tf,
		len:   len(terms),
	})

	// update avg doc length
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

// --- Vector Index (optional, for when embeddings are available) ---

type VectorIndex struct {
	vectors []vectorEntry
	dim     int
}

type vectorEntry struct {
	path   string
	chunk  string
	vector []float32
}

func NewVectorIndex(dim int) *VectorIndex {
	return &VectorIndex{dim: dim}
}

func (vi *VectorIndex) Add(path, chunk string, vec []float32) {
	vi.vectors = append(vi.vectors, vectorEntry{
		path:   path,
		chunk:  chunk,
		vector: vec,
	})
}

func (vi *VectorIndex) Search(query []float32, k int) []Result {
	type scored struct {
		idx   int
		score float64
	}

	var scores []scored
	for i, v := range vi.vectors {
		sim := cosineSim(query, v.vector)
		if sim > 0 {
			scores = append(scores, scored{i, sim})
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
			Path:  vi.vectors[s.idx].path,
			Score: s.score,
			Chunk: vi.vectors[s.idx].chunk,
		}
	}

	return results
}

func (vi *VectorIndex) Count() int {
	return len(vi.vectors)
}

func (vi *VectorIndex) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// write dim
	binary.Write(f, binary.LittleEndian, int32(vi.dim))
	// write count
	binary.Write(f, binary.LittleEndian, int32(len(vi.vectors)))

	for _, v := range vi.vectors {
		// write vector
		for _, val := range v.vector {
			binary.Write(f, binary.LittleEndian, val)
		}
	}

	return nil
}

func (vi *VectorIndex) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var dim, count int32
	binary.Read(f, binary.LittleEndian, &dim)
	binary.Read(f, binary.LittleEndian, &count)

	vi.dim = int(dim)
	vi.vectors = make([]vectorEntry, count)

	for i := range vi.vectors {
		vi.vectors[i].vector = make([]float32, dim)
		for j := range vi.vectors[i].vector {
			binary.Read(f, binary.LittleEndian, &vi.vectors[i].vector[j])
		}
	}

	return nil
}

// --- Utilities ---

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
	// chunk by lines, respecting size
	scanner := bufio.NewScanner(strings.NewReader(text))
	var chunks []string
	var current strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if current.Len()+len(line)+1 > size && current.Len() > 0 {
			chunks = append(chunks, current.String())
			// keep overlap
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

func cosineSim(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
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
		".r": true, ".lua": true, ".vim": true, ".el": true,
		".tex": true, ".bib": true, ".org": true, ".rst": true,
	}
	ext := strings.ToLower(filepath.Ext(name))
	return textExts[ext]
}

func isStopWord(w string) bool {
	stops := map[string]bool{
		"the": true, "is": true, "at": true, "of": true,
		"on": true, "in": true, "to": true, "and": true,
		"or": true, "an": true, "a": true, "it": true,
		"as": true, "be": true, "by": true, "do": true,
		"for": true, "from": true, "has": true, "he": true,
		"if": true, "no": true, "not": true, "she": true,
		"so": true, "that": true, "this": true, "was": true,
		"we": true, "with": true, "you": true, "are": true,
		"but": true, "had": true, "have": true, "his": true,
		"her": true, "its": true, "may": true, "my": true,
		"our": true, "than": true, "them": true, "they": true,
		"too": true, "very": true, "what": true, "when": true,
		"who": true, "will": true, "your": true,
	}
	return stops[w]
}

// Fmt is used to prevent import cycle
func Fmt(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}
