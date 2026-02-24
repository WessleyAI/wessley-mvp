package ingest

import (
	"strings"
	"unicode"
)

const (
	// DefaultChunkSize is the target number of tokens per chunk.
	DefaultChunkSize = 512
	// DefaultOverlap is the number of overlapping tokens between chunks.
	DefaultOverlap = 50
)

// splitSentences splits text into sentences using punctuation and newlines.
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			// Check it's end-of-sentence (next char is space/end or newline).
			if r == '\n' || i == len(text)-1 || (i+1 < len(text) && unicode.IsSpace(rune(text[i+1]))) {
				s := strings.TrimSpace(current.String())
				if s != "" {
					sentences = append(sentences, s)
				}
				current.Reset()
			}
		}
	}
	// Remaining text.
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}

// chunkSentences groups sentences into chunks of ~chunkSize tokens with overlap.
// Token count is approximated as word count.
func chunkSentences(docID string, sentences []string, chunkSize, overlap int) []Chunk {
	if len(sentences) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if overlap < 0 {
		overlap = 0
	}

	var chunks []Chunk
	idx := 0
	start := 0

	for start < len(sentences) {
		var buf strings.Builder
		tokens := 0
		end := start

		for end < len(sentences) {
			words := wordCount(sentences[end])
			if tokens+words > chunkSize && tokens > 0 {
				break
			}
			if buf.Len() > 0 {
				buf.WriteRune(' ')
			}
			buf.WriteString(sentences[end])
			tokens += words
			end++
		}

		chunks = append(chunks, Chunk{
			Text:  buf.String(),
			Index: idx,
			DocID: docID,
		})
		idx++

		// Move start back by overlap amount.
		overlapTokens := 0
		newStart := end
		for newStart > start && overlapTokens < overlap {
			newStart--
			overlapTokens += wordCount(sentences[newStart])
		}
		if newStart == start {
			// Ensure forward progress.
			start = end
		} else {
			start = newStart
		}
	}
	return chunks
}

func wordCount(s string) int {
	return len(strings.Fields(s))
}
