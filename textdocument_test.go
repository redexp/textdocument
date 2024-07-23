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
			Range: textdocument.NewRange(0, 0, 1, 0),
			Check: "TESTqwer\n⌘xc",
		},
		{
			Range: textdocument.NewRange(0, 0, 2, 3),
			Check: "TEST",
		},
		{
			Range: textdocument.NewRange(2, 3, 2, 3),
			Check: "⌘sd\nqwer\n⌘xcTEST",
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
		{0, 0, 0, 0},
		{0, 2, 4, 0},
		{0, 4, 6, 1},
		{1, 0, 6, 0},
		{1, 2, 8, 0},
		{1, 5, 11, 1},
		{2, 0, 11, 0},
		{2, 3, 16, 0},
		{2, 4, 17, 1},
		{3, 0, 0, 1},
	}

	for i, item := range list {
		index, err := doc.PositionToByteIndex(&proto.Position{
			Line:      item[0],
			Character: item[1],
		})

		if item[3] == 1 {
			if err == nil {
				t.Errorf("%d should return error", i)
			}
			continue
		}

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
		{16, 2, 3},
		{17, 3, 0},
	}

	for i, item := range list {
		if i == 6 {
			doc.SetText(doc.Text + "\n")
		}

		pos, err := doc.ByteIndexToPosition(item[0])

		if err != nil {
			t.Errorf("%d err %s", i, err)
		}

		if pos.Line != item[1] || pos.Character != item[2] {
			t.Errorf("%d pos (%d, %d) expected (%d, %d)", i, pos.Line, pos.Character, item[1], item[2])
		}
	}
}

func TestPointToPosition(t *testing.T) {
	doc := getDoc()

	list := [][]uint32{
		{0, 0, 0, 0},
		{0, 3, 0, 1},
		{1, 0, 1, 0},
		{1, 2, 1, 2},
		{2, 0, 2, 0},
		{2, 4, 2, 2},
	}

	for i, item := range list {
		pos, err := doc.PointToPosition(textdocument.Point{
			Row:    item[0],
			Column: item[1],
		})

		if err != nil {
			t.Errorf("%d err: %s", i, err)
		}

		if pos.Line != item[2] {
			t.Errorf("%d pos.Line %d expect %d", i, pos.Line, item[2])
		}

		if pos.Character != item[3] {
			t.Errorf("%d pos.Character %d expect %d", i, pos.Character, item[3])
		}
	}
}

func TestLineByteIndexToPosition(t *testing.T) {
	doc := getDoc()

	list := [][]uint32{
		{0, 0, 0, 0, 0},
		{0, 3, 0, 1, 0},
		{0, 4, 0, 2, 0},
		{0, 5, 0, 3, 0},
		{1, 0, 1, 0, 0},
		{1, 2, 1, 2, 0},
		{1, 5, 0, 0, 1},
		{2, 3, 2, 1, 0},
		{2, 4, 2, 2, 0},
		{2, 5, 2, 3, 0},
		{0, 6, 0, 0, 1},
		{2, 6, 0, 0, 1},
	}

	for i, item := range list {
		pos, err := doc.LineByteIndexToPosition(item[0], item[1])

		if item[4] == 1 {
			if err == nil {
				t.Errorf("%d should be error but it returns %v for {%d, %d}", i, pos, item[0], item[1])
			}
			continue
		}

		if err != nil {
			t.Errorf("%d err: %s", i, err)
			continue
		}

		if pos.Line != item[2] || pos.Character != item[3] {
			t.Errorf("%d wrong pos %v expect {%d, %d}", i, pos, item[2], item[3])
		}
	}
}

func TestGetNonSpaceTextAroundPosition(t *testing.T) {
	doc := textdocument.NewTextDocument("asd\nwer zxc")

	type Test struct {
		Line uint32
		Char uint32
		Text string
	}

	list := []Test{
		{
			Line: 0,
			Char: 0,
			Text: "asd",
		},
		{
			Line: 0,
			Char: 1,
			Text: "asd",
		},
		{
			Line: 1,
			Char: 0,
			Text: "wer",
		},
		{
			Line: 1,
			Char: 1,
			Text: "wer",
		},
		{
			Line: 1,
			Char: 3,
			Text: "wer",
		},
		{
			Line: 1,
			Char: 4,
			Text: "zxc",
		},
		{
			Line: 1,
			Char: 5,
			Text: "zxc",
		},
		{
			Line: 1,
			Char: 7,
			Text: "zxc",
		},
	}

	for i, item := range list {
		text, err := doc.GetNonSpaceTextAroundPosition(&textdocument.Position{
			Line:      item.Line,
			Character: item.Char,
		})

		if err != nil {
			t.Errorf("%d err: %s", i, err)
			continue
		}

		if text != item.Text {
			t.Errorf("%d wrong text '%s' expected '%s'", i, text, item.Text)
		}
	}
}
