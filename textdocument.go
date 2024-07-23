package textdocument

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	sitter "github.com/smacker/go-tree-sitter"
	proto "github.com/tliron/glsp/protocol_3_16"
)

func NewTextDocument(text string) *TextDocument {
	doc := TextDocument{
		Text: text,
	}

	doc.UpdateLines()

	return &doc
}

type TextDocument struct {
	Text       string
	TextLength UInt
	Lines      []UInt
	Tree       *sitter.Tree
	Parser     *sitter.Parser

	lastLineOffset lineOffsetColumn
}

type lineOffsetColumn struct {
	line   UInt
	offset UInt
	column UInt
}

type (
	UInt        = proto.UInteger
	ChangeEvent = proto.TextDocumentContentChangeEvent
	Position    = proto.Position
	Range       = proto.Range
	Point       = sitter.Point
)

func (doc *TextDocument) Change(e *ChangeEvent) error {
	return doc.ChangeCtx(e, nil)
}

func (doc *TextDocument) ChangeCtx(e *ChangeEvent, ctx *context.Context) error {
	start, err := doc.PositionToByteIndex(&e.Range.Start)

	if err != nil {
		return err
	}

	end, err := doc.PositionToByteIndex(&e.Range.End)

	if err != nil {
		return err
	}

	startPoint, err := doc.PositionToPoint(&e.Range.Start)

	if err != nil {
		return err
	}

	oldEndPoint, err := doc.PositionToPoint(&e.Range.End)

	if err != nil {
		return err
	}

	doc.Text = doc.Text[:start] + e.Text + doc.Text[end:]
	doc.UpdateLines()

	if doc.Tree == nil {
		return doc.UpdateTree(ctx)
	}

	newEndIndex := start + UInt(len(e.Text))
	newEndPoint, err := doc.ByteIndexToPoint(newEndIndex)

	if err != nil {
		return err
	}

	doc.Tree.Edit(sitter.EditInput{
		StartIndex:  start,
		OldEndIndex: end,
		NewEndIndex: newEndIndex,
		StartPoint:  *startPoint,
		OldEndPoint: *oldEndPoint,
		NewEndPoint: *newEndPoint,
	})

	return doc.UpdateTree(ctx)
}

func NewRange(startLine UInt, startChar UInt, endLine UInt, endChar UInt) *Range {
	return &Range{
		Start: Position{
			Line:      startLine,
			Character: startChar,
		},
		End: Position{
			Line:      endLine,
			Character: endChar,
		},
	}
}

func (doc *TextDocument) UpdateLines() {
	lines := strings.Split(doc.Text, "\n")
	doc.Lines = make([]UInt, len(lines))
	doc.TextLength = UInt(len(doc.Text))
	doc.lastLineOffset = lineOffsetColumn{}
	offset := UInt(0)

	for i, line := range lines {
		doc.Lines[i] = offset
		offset += 1 + UInt(len(line))
	}
}

// Same as SetTextCtx with ctx = nil
func (doc *TextDocument) SetText(text string) error {
	return doc.SetTextCtx(text, nil)
}

// Set Text, call UpdateLines() and UpdateTree(), be aware of how UpdateTree() will generate new Tree
func (doc *TextDocument) SetTextCtx(text string, ctx *context.Context) error {
	doc.Text = text
	doc.UpdateLines()

	return doc.UpdateTree(ctx)
}

// Same as SetParserCtx() with ctx = nil
func (doc *TextDocument) SetParser(parser *sitter.Parser) error {
	return doc.SetParserCtx(parser, nil)
}

// Will set Parser and call UpdateTree()
func (doc *TextDocument) SetParserCtx(parser *sitter.Parser, ctx *context.Context) error {
	doc.Parser = parser

	return doc.UpdateTree(ctx)
}

// Will update Tree. If Tree present and NOT changed then it will be fully regenerated.
// If Tree has changes then it will be used to generate new Tree
func (doc *TextDocument) UpdateTree(ctx *context.Context) error {
	if doc.Parser == nil {
		return nil
	}

	oldTree := doc.Tree

	if doc.Tree != nil && !doc.Tree.RootNode().HasChanges() {
		doc.Tree = nil
	}

	if ctx == nil {
		c := context.Background()
		ctx = &c
	}

	tree, err := doc.Parser.ParseCtx(*ctx, oldTree, []byte(doc.Text))

	if err != nil {
		doc.Tree = oldTree
		return err
	}

	doc.Tree = tree

	return nil
}

func (doc *TextDocument) PositionToByteIndex(pos *Position) (UInt, error) {
	linesCount := UInt(len(doc.Lines))

	if pos.Line >= linesCount {
		return 0, fmt.Errorf("line %d is out of range (%d)", pos.Line, linesCount-1)
	}

	character := UInt(0)
	offset := doc.Lines[pos.Line]
	max := doc.TextLength

	if pos.Line+1 < linesCount {
		max = doc.Lines[pos.Line+1] - 1
	}

	for character < pos.Character {
		char, size := utf8.DecodeRuneInString(doc.Text[offset:])

		if char == utf8.RuneError {
			return 0, errors.New("rune error")
		}

		offset += UInt(size)
		character++

		if offset > max || (offset == max && character < pos.Character) {
			return 0, fmt.Errorf("character %d is out of reange (%d) for line %d", pos.Character, character, pos.Line)
		}
	}

	return offset, nil
}

func (doc *TextDocument) ByteIndexLine(index UInt) (UInt, error) {
	if index > doc.TextLength {
		return 0, fmt.Errorf("byte index %d is out of range (%d)", index, doc.TextLength)
	}

	line := UInt(len(doc.Lines) - 1)

	for {
		if line == 0 || doc.Lines[line] <= index {
			break
		}

		line--
	}

	return line, nil
}

// byte index means number of bytes from text start
func (doc *TextDocument) ByteIndexToPosition(index UInt) (*Position, error) {
	line, err := doc.ByteIndexLine(index)

	if err != nil {
		return nil, err
	}

	offset := doc.Lines[line]

	return doc.LineByteIndexToPosition(line, index-offset)
}

func (doc *TextDocument) ByteIndexToPoint(index UInt) (*Point, error) {
	line, err := doc.ByteIndexLine(index)

	if err != nil {
		return nil, err
	}

	offset := doc.Lines[line]

	return &Point{
		Row:    line,
		Column: index - offset,
	}, nil
}

// index is number of bytes from line start
func (doc *TextDocument) LineByteIndexToPosition(line UInt, index UInt) (*Position, error) {
	offset, max, err := doc.LineMinMaxByteIndex(line)

	if err != nil {
		return nil, err
	}

	column := UInt(0)
	index += offset
	last := &doc.lastLineOffset

	if last.line == line && last.offset <= index {
		offset = last.offset
		column = last.column
	}

	for {
		if offset >= index {
			break
		}

		char, size := utf8.DecodeRuneInString(doc.Text[offset:])

		if char == utf8.RuneError {
			return nil, errors.New("rune error")
		}

		offset += UInt(size)
		column++

		if offset > max {
			return nil, fmt.Errorf("byte index %d is out of reange (%d) for line %d", index-doc.Lines[line], max-doc.Lines[line], line)
		}
	}

	last.line = line
	last.offset = offset
	last.column = column

	return &Position{
		Line:      line,
		Character: column,
	}, nil
}

func (doc *TextDocument) PointToPosition(point Point) (*Position, error) {
	return doc.LineByteIndexToPosition(point.Row, point.Column)
}

func (doc *TextDocument) PositionToPoint(pos *Position) (*Point, error) {
	index, err := doc.PositionToByteIndex(pos)

	if err != nil {
		return nil, err
	}

	offset := doc.Lines[pos.Line]

	return &Point{
		Row:    pos.Line,
		Column: index - offset,
	}, nil
}

func (doc *TextDocument) LineMinMaxByteIndex(line UInt) (UInt, UInt, error) {
	linesCount := UInt(len(doc.Lines))

	if line >= linesCount {
		return 0, 0, fmt.Errorf("line %d is out of range (%d)", line, linesCount)
	}

	min := doc.Lines[line]
	max := doc.TextLength

	if line+1 < linesCount {
		max = doc.Lines[line+1] - 1
	}

	return min, max, nil
}

func (doc *TextDocument) GetNonSpaceTextAroundPosition(pos *Position) (string, error) {
	end, err := doc.PositionToByteIndex(pos)

	if err != nil {
		return "", err
	}

	start := end
	min, max, err := doc.LineMinMaxByteIndex(pos.Line)

	if err != nil {
		return "", err
	}

	for {
		if start <= min {
			start = min
			break
		}

		char, size := utf8.DecodeLastRuneInString(doc.Text[min:start])

		if char == utf8.RuneError {
			return "", errors.New("rune error")
		}

		if char == ' ' {
			break
		}

		start -= UInt(size)
	}

	for {
		if end >= max {
			end = max
			break
		}

		char, size := utf8.DecodeRuneInString(doc.Text[end:max])

		if char == utf8.RuneError {
			return "", errors.New("rune error")
		}

		if char == ' ' {
			break
		}

		end += UInt(size)
	}

	return doc.Text[start:end], nil
}
