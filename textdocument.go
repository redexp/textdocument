package textdocument

import (
	"context"
	"errors"
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
	Text   string
	Lines  []uint
	Tree   *sitter.Tree
	Parser *sitter.Parser
}

type uint = proto.UInteger

func (doc *TextDocument) Change(e *proto.TextDocumentContentChangeEvent) error {
	return doc.ChangeCtx(e, nil)
}

func (doc *TextDocument) ChangeCtx(e *proto.TextDocumentContentChangeEvent, ctx *context.Context) error {
	start, err := doc.PositionToByteIndex(&e.Range.Start)

	if err != nil {
		return err
	}

	end, err := doc.PositionToByteIndex(&e.Range.End)

	if err != nil {
		return err
	}

	doc.SetText(doc.Text[:start] + e.Text + doc.Text[end:])

	if doc.Tree == nil {
		return nil
	}

	endIndex := start + uint(len(e.Text))
	endPos, err := doc.ByteIndexToPosition(endIndex)

	if err != nil {
		return err
	}

	doc.Tree.Edit(sitter.EditInput{
		StartIndex:  start,
		OldEndIndex: end,
		NewEndIndex: endIndex,
		StartPoint:  PositionToPoint(&e.Range.Start),
		OldEndPoint: PositionToPoint(&e.Range.End),
		NewEndPoint: PositionToPoint(endPos),
	})

	if doc.Parser == nil {
		return nil
	}

	return doc.UpdateTree(ctx)
}

func NewRange(startLine uint, startChar uint, endLine uint, endChar uint) *proto.Range {
	return &proto.Range{
		Start: proto.Position{
			Line:      startLine,
			Character: startChar,
		},
		End: proto.Position{
			Line:      endLine,
			Character: endChar,
		},
	}
}

func PositionToPoint(pos *proto.Position) sitter.Point {
	return sitter.Point{
		Row:    pos.Line,
		Column: pos.Character,
	}
}

func (doc *TextDocument) UpdateLines() {
	lines := strings.Split(doc.Text, "\n")
	doc.Lines = make([]uint, len(lines))
	offset := uint(0)

	for i, line := range lines {
		doc.Lines[i] = offset
		offset += 1 + uint(len(line))
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

func (doc *TextDocument) PositionToByteIndex(pos *proto.Position) (uint, error) {
	linesCount := uint(len(doc.Lines))

	if pos.Line >= linesCount {
		return 0, errors.New("out of range")
	}

	offset := doc.Lines[pos.Line]

	for i := uint(0); i < pos.Character; i++ {
		char, size := utf8.DecodeRuneInString(doc.Text[offset:])

		if char == utf8.RuneError {
			return 0, errors.New("rune error")
		}

		offset += uint(size)
	}

	return offset, nil
}

func (doc *TextDocument) ByteIndexToPosition(index uint) (*proto.Position, error) {
	if index >= uint(len(doc.Text)) {
		return nil, errors.New("out of range")
	}

	count := uint(len(doc.Lines) - 1)
	line := count

	for ; line >= 0; line-- {
		if doc.Lines[line] <= index {
			break
		}
	}

	column := uint(0)
	offset := doc.Lines[line]

	for {
		if offset >= index {
			break
		}

		char, size := utf8.DecodeRuneInString(doc.Text[offset:])

		if char == utf8.RuneError {
			return nil, errors.New("rune error")
		}

		offset += uint(size)
		column++
	}

	return &proto.Position{
		Line:      line,
		Character: column,
	}, nil
}
