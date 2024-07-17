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

	err = doc.SetText(doc.Text[:start] + e.Text + doc.Text[end:])

	if err != nil {
		return err
	}

	if doc.Tree == nil {
		return nil
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

	if doc.Parser == nil {
		return nil
	}

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

func (doc *TextDocument) SetText(text string) error {
	return doc.SetTextCtx(text, nil)
}

func (doc *TextDocument) SetTextCtx(text string, ctx *context.Context) error {
	doc.Text = text
	doc.UpdateLines()

	if doc.Parser == nil {
		return nil
	}

	return doc.UpdateTree(ctx)
}

func (doc *TextDocument) SetParser(parser *sitter.Parser) error {
	return doc.SetParserCtx(parser, nil)
}

func (doc *TextDocument) SetParserCtx(parser *sitter.Parser, ctx *context.Context) error {
	doc.Parser = parser

	if doc.Tree != nil {
		return nil
	}

	return doc.UpdateTree(ctx)
}

func (doc *TextDocument) UpdateTree(ctx *context.Context) error {
	var tree *sitter.Tree
	var err error

	if ctx != nil {
		tree, err = doc.Parser.ParseCtx(*ctx, doc.Tree, []byte(doc.Text))
	} else {
		tree = doc.Parser.Parse(doc.Tree, []byte(doc.Text))
	}

	if err != nil {
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
	if line >= UInt(len(doc.Lines)) {
		return nil, fmt.Errorf("line %d is out of range (%d)", line, len(doc.Lines))
	}

	column := UInt(0)
	offset := doc.Lines[line]
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
