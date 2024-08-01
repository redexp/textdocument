package textdocument_test

import (
	"testing"

	"github.com/redexp/textdocument"
	sitter "github.com/smacker/go-tree-sitter"
	js "github.com/smacker/go-tree-sitter/javascript"
	proto "github.com/tliron/glsp/protocol_3_16"
)

func getDoc() *textdocument.TextDocument {
	return textdocument.NewTextDocument("⌘sd\nqwer\n⌘xc") // 5 n 4 n 5 = 16
}

func createParser() *sitter.Parser {
	p := sitter.NewParser()
	p.SetLanguage(getLang())
	return p
}

func getLang() *sitter.Language {
	return js.GetLanguage()
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

func TestGetNodesByRange(t *testing.T) {
	text := "var x = 1\nvar y = 2\nvar z = 3"
	doc := textdocument.NewTextDocument(text)
	doc.SetParser(createParser())

	list := []struct {
		StartLine uint32
		StartChar uint32
		EndLine   uint32
		EndChar   uint32
		Values    []string
	}{
		{0, 4, 0, 9, []string{"x = 1"}},
		{0, 1, 0, 5, []string{"var", "x"}},
		{0, 8, 2, 1, []string{"1", "var y = 2", "var"}},
		{1, 0, 1, 9, []string{"var y = 2"}},
		{1, 0, 2, 0, []string{"var y = 2"}},
		{2, 8, 2, 9, []string{"3"}},
	}

	for i, item := range list {
		start := proto.Position{
			Line:      item.StartLine,
			Character: item.StartChar,
		}
		end := proto.Position{
			Line:      item.EndLine,
			Character: item.EndChar,
		}
		nodes, err := doc.GetNodesByRange(&start, &end)

		if err != nil {
			t.Errorf("%d err: %s", i, err)
			continue
		}

		values := make([]string, len(nodes))

		for i, node := range nodes {
			values[i] = node.Content([]byte(text))
		}

		if len(values) != len(item.Values) {
			t.Errorf("%d values: %v expect %v", i, values, item.Values)
			continue
		}

		for j, value := range item.Values {
			if values[j] != value {
				t.Errorf("%d:%d value: '%s' expect '%s'", i, j, values[j], value)
			}
		}
	}
}

func TestGetNodeByPosition(t *testing.T) {
	text := "var x = 1\nvar y =  2\nvar z = 3"
	doc := textdocument.NewTextDocument(text)
	doc.SetParser(createParser())

	list := []struct {
		StartLine uint32
		StartChar uint32
		Value     string
	}{
		{0, 4, "x"},
		{0, 1, "var"},
		{0, 8, "1"},
		{1, 0, "var"},
		{1, 5, "y"},
		{1, 8, ""},
		{2, 9, "3"},
	}

	for i, item := range list {
		start := proto.Position{
			Line:      item.StartLine,
			Character: item.StartChar,
		}
		node, err := doc.GetNodeByPosition(&start)

		if err != nil {
			t.Errorf("%d err: %s", i, err)
			continue
		}

		if node == nil {
			if item.Value == "" {
				continue
			}

			t.Errorf("%d node nil, pos: %v", i, item)
			continue
		}

		value := node.Content([]byte(text))

		if item.Value != value {
			t.Errorf("%d value: '%s' expect '%s'", i, value, item.Value)
		}
	}
}

func TestHighlights(t *testing.T) {
	doc := textdocument.NewTextDocument("var x = 1\nvar y = 2\nvar zxc = 3")
	doc.SetParser(createParser())

	pattern := "(identifier) @ident\n(number) @num"
	q, _ := sitter.NewQuery([]byte(pattern), getLang())
	doc.SetHighlightQuery(q, &textdocument.Ignore{
		Missing: true,
		Extra:   true,
	})

	if len(doc.HighlightCaptures) != 6 {
		t.Errorf("init HighlightCaptures wrong len %d expect %d", len(doc.HighlightCaptures), 6)
	}

	capTests := []struct {
		Line  uint32
		Char  uint32
		Index uint32
		Value string
	}{
		{0, 8, 1, "1"},
		{1, 4, 0, "y"},
		{2, 11, 1, "3"},
	}

	for i, item := range capTests {
		cap, err := doc.GetHighlightCaptureByPosition(&textdocument.Position{
			Line:      item.Line,
			Character: item.Char,
		})

		if err != nil {
			t.Errorf("%d err %s", i, err)
			continue
		}

		if cap.Index != item.Index {
			t.Errorf("%d cap wrong Index %d expect %d", i, cap.Index, item.Index)
		}

		str := cap.Node.Content([]byte(doc.Text))

		if str != item.Value {
			t.Errorf("%d cap.Node.Content '%s' expect '%s'", i, str, item.Value)
		}
	}

	closestTests := []struct {
		Line   uint32
		Char   uint32
		Prev   string
		Target string
		Next   string
	}{
		{0, 0, "", "", "x"},
		{0, 4, "", "x", "1"},
		{0, 5, "", "x", "1"},
		{0, 6, "x", "", "1"},
		{0, 8, "x", "1", "y"},
		{0, 9, "x", "1", "y"},
		{1, 0, "1", "", "y"},
		{2, 11, "zxc", "3", ""},
	}

	for i, item := range closestTests {
		prev, target, next, err := doc.GetClosestHighlightCaptureByPosition(&textdocument.Position{
			Line:      item.Line,
			Character: item.Char,
		})

		if err != nil {
			t.Errorf("%d err %s", i, err)
			continue
		}

		caps := []*sitter.QueryCapture{prev, target, next}
		values := []string{item.Prev, item.Target, item.Next}

		for n, cap := range caps {
			if cap == nil {
				if values[n] != "" {
					t.Errorf("%d cap %d is nil expect '%s'", i, n, values[n])
					break
				}

				continue
			}

			value := cap.Node.Content([]byte(doc.Text))

			if value != values[n] {
				t.Errorf("%d cap %d is '%s' expect '%s'", i, n, value, values[n])
				break
			}
		}
	}

	list := []struct {
		Pos  []uint32
		Text string
	}{
		{[]uint32{0, 4, 0, 5}, "z"},
		{[]uint32{1, 8, 1, 9}, "4"},
		{[]uint32{2, 5, 2, 11}, "cx = 5"},
		{[]uint32{0, 7, 0, 7}, "  "},
		{[]uint32{2, 4, 2, 5}, ""},
	}

	for i, item := range list {
		start := &proto.Position{
			Line:      item.Pos[0],
			Character: item.Pos[1],
		}
		end := &proto.Position{
			Line:      item.Pos[2],
			Character: item.Pos[3],
		}

		err := doc.Change(&textdocument.ChangeEvent{
			Range: &proto.Range{
				Start: *start,
				End:   *end,
			},
			Text: item.Text,
		})

		if err != nil {
			t.Errorf("%d err %s", i, err)
		}
	}

	legend := textdocument.HighlightLegend{
		{
			Type:      0,
			Modifiers: 0,
		},
		{
			Type:      1,
			Modifiers: 1,
		},
	}

	tags, err := doc.ConvertHighlightCaptures(legend)

	if err != nil {
		t.Error(err)
		return
	}

	comp := []uint32{
		0, 4, 1, 0, 0,
		0, 6, 1, 1, 1,
		1, 4, 1, 0, 0,
		0, 4, 1, 1, 1,
		1, 4, 2, 0, 0,
		0, 5, 1, 1, 1,
	}

	count := len(comp)

	if len(tags) != count {
		t.Errorf("tags len %d expected %d", len(tags), count)
		return
	}

	for i := 0; i < count; i += 5 {
		for n := 0; n < 5; n++ {
			if tags[i+n] != comp[i+n] {
				t.Errorf("%d wrong tag %v expected %v\n", i/5, tags[i:i+5], comp[i:i+5])
				return
			}
		}
	}
}
