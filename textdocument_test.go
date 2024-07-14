package textdocument_test

import (
	"testing"

	"github.com/redexp/textdocument"
	proto "github.com/tliron/glsp/protocol_3_16"
)

func getDoc() *textdocument.TextDocument {
	return textdocument.NewTextDocument("⌘sd\nqwer\n⌘xc") // 5 n 4 n 5 = 16
}

func TestUpdateLines(t *testing.T) {
	doc := getDoc()

	if len(doc.Lines) != 3 {
		t.Errorf("Lines should be len 3, actual %d", len(doc.Lines))
	}

	lines := []uint32{0, 6, 11}

	for i, offset := range lines {
		if doc.Lines[i] != offset {
			t.Errorf("%d line wrong offset %d, expect %d", i, doc.Lines[i], offset)
		}
	}
}

func TestChange(t *testing.T) {
	doc := getDoc()

	list := []struct {
		Range *proto.Range
		Text  string
		Check string
	}{
		{
			Range: textdocument.NewRange(0, 0, 2, 1),
			Check: "TESTxc",
		},
		{
			Range: textdocument.NewRange(0, 0, 0, 1),
			Check: "TESTsd\nqwer\n⌘xc",
		},
		{
			Range: textdocument.NewRange(1, 1, 1, 1),
			Check: "⌘sd\nqTESTwer\n⌘xc",
		},
		{
			Range: textdocument.NewRange(0, 0, 2, 3),
			Check: "TEST",
		},
	}

	reset := doc.Text

	for i, item := range list {
		doc.SetText(reset)

		text := item.Text

		if text == "" {
			text = "TEST"
		}

		err := doc.Change(&proto.TextDocumentContentChangeEvent{
			Range: item.Range,
			Text:  text,
		})

		if err != nil {
			t.Errorf("%d - doc.Change err %s", i, err.Error())
		}

		if doc.Text != item.Check {
			t.Errorf("%d - %s expect %s", i, doc.Text, item.Check)
		}
	}
}

func TestPositionToByteIndex(t *testing.T) {
	doc := getDoc()

	list := [][]uint32{
		{0, 0, 0},
		{0, 2, 4},
		{1, 0, 6},
		{1, 2, 8},
	}

	for i, item := range list {
		index, err := doc.PositionToByteIndex(&proto.Position{
			Line:      item[0],
			Character: item[1],
		})

		if err != nil {
			t.Errorf("PositionToByteIndex err: %s", err.Error())
		}

		if index != item[2] {
			t.Errorf("%d index %d expect %d", i, index, item[2])
		}
	}
}

func TestByteIndexToPosition(t *testing.T) {
	doc := getDoc()

	list := [][]uint32{
		{0, 0, 0},
		{3, 0, 1},
		{4, 0, 2},
		{7, 1, 1},
		{15, 2, 2},
	}

	for i, item := range list {
		pos, err := doc.ByteIndexToPosition(item[0])

		if err != nil {
			t.Errorf("%d err %s", i, err)
		}

		if pos.Line != item[1] || pos.Character != item[2] {
			t.Errorf("%d pos (%d, %d) expected (%d, %d)", i, pos.Line, pos.Character, item[1], item[2])
		}
	}
}
