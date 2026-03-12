package escpos

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type xmlRawItem struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",chardata"`
}

type xmlEPOSPrint struct {
	XMLName xml.Name     `xml:"epos-print"`
	Items   []xmlRawItem `xml:",any"`
}

func ParseXML(body []byte) ([]byte, error) {
	s := string(body)
	start := strings.Index(s, "<epos-print")
	end := strings.LastIndex(s, "</epos-print>")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no <epos-print> element found in request body")
	}
	fragment := s[start : end+len("</epos-print>")]

	var ep xmlEPOSPrint
	if err := xml.Unmarshal([]byte(fragment), &ep); err != nil {
		return nil, fmt.Errorf("XML parse error: %w", err)
	}

	job := append([]byte(nil), CmdInit...)

	for _, item := range ep.Items {
		tag := strings.ToLower(item.XMLName.Local)
		attrs := attrMap(item.Attrs)

		switch tag {
		case "text":
			job = append(job, BuildText(item.Content, textAttrsFromMap(attrs))...)

		case "feed":
			lines := clamp(parseInt(attrs["line"], 1), 1, 255)
			for i := 0; i < lines; i++ {
				job = append(job, LF)
			}

		case "cut":
			job = append(job, CmdCut...)

		case "pulse":
			job = append(job, CmdPulse...)

		case "image":
			imgAttrs := ImageAttrs{
				Align:  attrs["align"],
				Width:  parseInt(attrs["width"], 0),
				Height: parseInt(attrs["height"], 0),
			}
			imgCmd, err := BuildImage(strings.TrimSpace(item.Content), imgAttrs)
			if err != nil {
				return nil, fmt.Errorf("image element: %w", err)
			}
			job = append(job, imgCmd...)

		default:
			return nil, fmt.Errorf("unsupported element <%s> inside <epos-print>", tag)
		}
	}

	return job, nil
}

func attrMap(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[strings.ToLower(a.Name.Local)] = a.Value
	}
	return m
}

func parseInt(s string, def int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return def
	}
	return n
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func boolPtr(m map[string]string, key string) *bool {
	switch m[key] {
	case "1", "true":
		t := true
		return &t
	case "0", "false":
		f := false
		return &f
	}
	return nil
}

func textAttrsFromMap(m map[string]string) TextAttrs {
	return TextAttrs{
		Align:        m["align"],
		Font:         m["font"],
		Em:           boolPtr(m, "em"),
		Underline:    boolPtr(m, "ul"),
		DoubleWidth:  boolPtr(m, "dw"),
		DoubleHeight: boolPtr(m, "dh"),
	}
}
