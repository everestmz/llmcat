package treesym

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/everestmz/llmcat/treesym/language"
	"github.com/everestmz/llmcat/treesym/tags"
	sitter "github.com/smacker/go-tree-sitter"
)

type SourceFile struct {
	Path string
	Text string
}

type Node struct {
	sitter.Range
	SummaryEndPoint sitter.Point
	Name            string
	Summary         string
	FullText        string
	Kind            string
	Documentation   string
}

type Symbols struct {
	Definitions []*Node
	References  []*Node
}

type OutlineChunk struct {
	// Name only set if item is omitted, since otherwise chunk could be bigger than a single symbol
	Name       string
	Content    string
	ShouldOmit bool
	// 0-indexed, like tree-sitter rows are
	StartRow int
	EndRow   int

	hasLines bool
}

func (oc *OutlineChunk) AddLine(line string) {
	if oc.hasLines {
		oc.Content += "\n"
	} else {
		oc.hasLines = true
	}

	oc.Content += line
}

type ProcessedSourceFile struct {
	SourceFile
	Symbols
}

// GetOutline runs through all of the definitions in the outline, and returns
// a list of chunks that represent contiguous blocks of code. If a chunk should
// be omitted for summarization, ShouldOmit is true, but the chunk is still
// returned, so that omitted items can be easily expanded
func (psf *ProcessedSourceFile) GetOutline() []*OutlineChunk {
	lines := strings.Split(psf.Text, "\n")

	var getLine = func(lineNum int) string {
		return lines[lineNum]
	}

	var summaryChunks []*OutlineChunk

	// In this first pass, we go through and create chunks only for omitted sections
	// We're also extending summaries to full line chunks, which are easier to manage
	for _, def := range psf.Definitions {
		if def.FullText == def.Summary {
			// Can't summarize
			continue
		}

		// We start on the line after the summary finishes
		startLine := int(def.SummaryEndPoint.Row) + 1
		endLine := int(def.EndPoint.Row)

		currentChunk := &OutlineChunk{
			ShouldOmit: true,
			StartRow:   startLine,
			EndRow:     endLine,
			Name:       def.Name,
		}

		for currentLine := startLine; currentLine <= endLine; currentLine++ {
			currentChunk.AddLine(getLine(currentLine))
		}
		summaryChunks = append(summaryChunks, currentChunk)
	}

	var chunks []*OutlineChunk

	currentChunk := &OutlineChunk{
		StartRow: 0,
	}
	var nextSummaryIdx = -1
	if len(summaryChunks) > 0 {
		nextSummaryIdx = 0
	}
	// Tree-sitter rows are 0-indexed
	for currentLine := 0; currentLine < len(lines); currentLine++ {
		if nextSummaryIdx == -1 {
			currentChunk.AddLine(getLine(currentLine))
			continue
		}

		nextSummary := summaryChunks[nextSummaryIdx]
		if currentLine != nextSummary.StartRow {
			currentChunk.AddLine(getLine(currentLine))
			continue
		}

		// We're in a summary block
		currentChunk.EndRow = currentLine - 1
		if len(currentChunk.Content) > 0 {
			chunks = append(chunks, currentChunk)
		}

		chunks = append(chunks, nextSummary)

		// +1 will bring us to the correct next line
		currentLine = nextSummary.EndRow

		currentChunk = &OutlineChunk{
			StartRow: currentLine + 1,
		}

		if len(summaryChunks) > nextSummaryIdx+1 {
			nextSummaryIdx += 1
		} else {
			nextSummaryIdx = -1
		}
	}

	return chunks
}

func GetSymbols(ctx context.Context, file *SourceFile) (*ProcessedSourceFile, error) {
	tsLang, err := GetTreeSitterLanguage(file.Path)
	if err != nil {
		return nil, err
	}

	parser, err := GetParser(file.Path)
	if err != nil {
		return nil, err
	}

	tree, err := parser.ParseCtx(ctx, nil, []byte(file.Text))
	if err != nil {
		return nil, fmt.Errorf("parsing code: %w", err)
	}

	ext := filepath.Ext(file.Path)

	lang, err := language.GetLanguage(ext)
	if err != nil {
		return nil, err
	}

	tagsQuery, err := tags.GetTagsQuery(lang)
	if err != nil {
		return nil, fmt.Errorf("running query: %w", err)
	}

	q, err := sitter.NewQuery([]byte(tagsQuery), tsLang)
	if err != nil {
		return nil, fmt.Errorf("creating query: %w", err)
	}

	qc := sitter.NewQueryCursor()

	qc.Exec(q, tree.RootNode())

	psf := &ProcessedSourceFile{
		SourceFile: *file,
		Symbols:    Symbols{},
	}

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		m = qc.FilterPredicates(m, []byte(file.Text))

		var docs string
		var nameCapture sitter.QueryCapture
		var contentCapture sitter.QueryCapture

		var numNonDocCaptures int
		for _, cap := range m.Captures {
			captureName := q.CaptureNameForId(cap.Index)
			if strings.HasPrefix(captureName, "name.") {
				nameCapture = cap
				numNonDocCaptures++
			} else if captureName == "doc" {
				docs = cap.Node.Content([]byte(file.Text))
			} else {
				contentCapture = cap
				numNonDocCaptures++
			}
		}

		if numNonDocCaptures != 2 {
			// XXX: not sure if this will always be 2. But for now, I think it is.
			var captureNames []string
			for _, cap := range m.Captures {
				captureNames = append(captureNames, q.CaptureNameForId(cap.Index))
			}
			return nil, fmt.Errorf("Got unexpected number of captures (%d) id:%d pattern_index:%d captures: %s", len(m.Captures), m.ID, m.PatternIndex, strings.Join(captureNames, ","))
		}

		captureName := q.CaptureNameForId(contentCapture.Index)
		if strings.HasPrefix(captureName, "name.") {
			continue
		}
		captureNameSplit := strings.Split(captureName, ".")
		captureType := captureNameSplit[0]
		captureSubType := captureNameSplit[1]

		node := &Node{
			Range:           contentCapture.Node.Range(),
			Name:            nameCapture.Node.Content([]byte(file.Text)),
			Summary:         contentCapture.Node.Content([]byte(file.Text)),
			Kind:            captureSubType,
			SummaryEndPoint: contentCapture.Node.EndPoint(),
			Documentation:   docs,
		}
		node.FullText = node.Summary

		if captureSubType != "function" && captureSubType != "method" {

			switch captureType {
			case "definition":
				psf.Symbols.Definitions = append(psf.Symbols.Definitions, node)
			case "reference":
				psf.Symbols.References = append(psf.Symbols.References, node)
			}
			continue
		}

		// This is a function or class - we can't pass the full body as the summary
		nodeStart := contentCapture.Node.StartPoint()
		var curMaxNode *sitter.Node
		// Basically what we're doing here is finding the largest node that starts
		// on the first line of this definition, but doesn't end on the last line.
		// We can capture multi-line function arg defs that way, but not the
		// whole function. Some of our captures capture the whole function
		for i := 0; i < int(contentCapture.Node.ChildCount()); i++ {
			child := contentCapture.Node.Child(i)
			if child.StartPoint().Row != nodeStart.Row {
				continue
			}

			if child.EndPoint().Row == contentCapture.Node.EndPoint().Row {
				continue
			}

			if curMaxNode == nil {
				curMaxNode = child
			}

			if len(child.Content([]byte(file.Text))) > len(curMaxNode.Content([]byte(file.Text))) {
				curMaxNode = child
			}
		}
		if curMaxNode == nil || contentCapture.Node.StartPoint().Row == contentCapture.Node.EndPoint().Row {
			continue
		}
		startByte := contentCapture.Node.StartByte()
		endByte := seekNewLine(file.Text, curMaxNode.EndByte(), 100)
		node.Summary = file.Text[startByte:endByte]
		node.SummaryEndPoint = curMaxNode.EndPoint()

		psf.Symbols.Definitions = append(psf.Symbols.Definitions, node)
	}

	return psf, nil
}

func seekNewLine(text string, startIndex uint32, maxSeekLength uint32) uint32 {
	for i := uint32(0); i < maxSeekLength; i++ {
		index := startIndex + i
		if index >= uint32(len(text)) {
			return uint32(len(text))
		}
		if text[index] == '\n' {
			return index
		}
	}

	return min(startIndex+maxSeekLength, uint32(len(text)))
}
