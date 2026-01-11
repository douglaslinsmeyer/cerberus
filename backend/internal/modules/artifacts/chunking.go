package artifacts

import (
	"strings"
	"unicode"
)

// Chunk represents a portion of a document
type Chunk struct {
	Index       int
	Text        string
	StartOffset int
	EndOffset   int
	TokenCount  int
}

// ChunkingStrategy defines parameters for document chunking
type ChunkingStrategy struct {
	MaxTokens          int  // Target tokens per chunk (4000-8000)
	OverlapTokens      int  // Token overlap between chunks (200)
	PreserveStructure  bool // Try to break at natural boundaries
}

// DefaultChunkingStrategy returns the recommended chunking configuration
func DefaultChunkingStrategy() *ChunkingStrategy {
	return &ChunkingStrategy{
		MaxTokens:         6000, // 6K tokens per chunk
		OverlapTokens:     200,  // 200 token overlap
		PreserveStructure: true,
	}
}

// ChunkDocument breaks a document into overlapping chunks
func (cs *ChunkingStrategy) ChunkDocument(content string) []Chunk {
	if content == "" {
		return []Chunk{}
	}

	// Estimate tokens using word-based approximation
	// 1 word ≈ 1.3 tokens (conservative)
	words := cs.splitIntoWords(content)
	totalTokens := int(float64(len(words)) * 1.3)

	// If document fits in one chunk, return as-is
	if totalTokens <= cs.MaxTokens {
		return []Chunk{{
			Index:       0,
			Text:        content,
			StartOffset: 0,
			EndOffset:   len(words),
			TokenCount:  totalTokens,
		}}
	}

	chunks := []Chunk{}
	wordsPerChunk := int(float64(cs.MaxTokens) / 1.3) // Convert tokens back to words
	overlapWords := int(float64(cs.OverlapTokens) / 1.3)

	start := 0
	chunkIndex := 0

	for start < len(words) {
		end := start + wordsPerChunk
		if end > len(words) {
			end = len(words)
		}

		// Try to find natural break if enabled
		if cs.PreserveStructure && end < len(words) {
			naturalEnd := cs.findNaturalBreak(words, start, end)
			if naturalEnd > start {
				end = naturalEnd
			}
		}

		// Get chunk text
		chunkWords := words[start:end]
		chunkText := strings.Join(chunkWords, " ")
		tokensEstimate := int(float64(len(chunkWords)) * 1.3)

		chunks = append(chunks, Chunk{
			Index:       chunkIndex,
			Text:        chunkText,
			StartOffset: start,
			EndOffset:   end,
			TokenCount:  tokensEstimate,
		})

		chunkIndex++

		// Move start position with overlap
		start = end - overlapWords
		if start < 0 {
			start = 0
		}

		// Prevent infinite loop
		if start >= len(words)-overlapWords {
			break
		}
	}

	return chunks
}

// splitIntoWords splits text into words
func (cs *ChunkingStrategy) splitIntoWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	})
}

// findNaturalBreak tries to find a paragraph or sentence boundary
func (cs *ChunkingStrategy) findNaturalBreak(words []string, start, end int) int {
	// Look in the last 10% of the chunk for a natural break
	searchStart := end - ((end - start) / 10)
	if searchStart < start {
		searchStart = start
	}

	// Look for paragraph breaks (words ending with \n\n)
	for i := end - 1; i >= searchStart; i-- {
		if strings.Contains(words[i], "\n\n") {
			return i + 1
		}
	}

	// Look for sentence breaks (words ending with . ! ?)
	for i := end - 1; i >= searchStart; i-- {
		word := strings.TrimSpace(words[i])
		if strings.HasSuffix(word, ".") ||
			strings.HasSuffix(word, "!") ||
			strings.HasSuffix(word, "?") {
			return i + 1
		}
	}

	// No natural break found, use original end
	return end
}

// EstimateTokens estimates token count from text using word-based approximation
func EstimateTokens(text string) int {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	})
	return int(float64(len(words)) * 1.3)
}
