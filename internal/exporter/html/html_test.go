package html

import (
	"strings"
	"testing"
)

func TestExportHTML(t *testing.T) {
	neotex := "Hello <neo> | 1:Fr; 3:HL:<https://example.com>"

	out, err := ExportHTML(neotex)
	if err != nil {
		t.Fatalf("ExportHTML returned error: %v", err)
	}

	if !strings.Contains(out, "id=\"neo-source\"") {
		t.Fatalf("neo-source block missing")
	}

	if !strings.Contains(out, "<pre id=\"neo-source\">") {
		t.Fatalf("neo-source block missing")
	}

	if !strings.Contains(out, "Hello &lt;neo&gt; | 1:Fr; 3:HL:&lt;https://example.com&gt;") {
		t.Fatalf("neotex content not escaped or missing")
	}

	if !strings.Contains(out, "data:font/woff;base64,") {
		t.Fatalf("font data URI missing")
	}

	if !strings.Contains(out, "parseNeotex") {
		t.Fatalf("decoder script missing")
	}

	if !strings.Contains(out, "ansi-link:hover") {
		t.Fatalf("hover rule missing")
	}
}

func TestExportHTMLPack(t *testing.T) {
	neotex := "Hello <neo> | 1:Fr; 3:HL:<https://example.com>"

	files, err := ExportHTMLPack(neotex)
	if err != nil {
		t.Fatalf("ExportHTMLPack returned error: %v", err)
	}

	if !strings.Contains(files.HTML, "style.css") || !strings.Contains(files.HTML, "app.js") {
		t.Fatalf("pack html missing references to assets")
	}
	if !strings.Contains(files.CSS, "@font-face") || !strings.Contains(files.CSS, FontFileName) {
		t.Fatalf("pack css missing font reference")
	}
	if !strings.Contains(files.JS, "parseNeotex") {
		t.Fatalf("pack js missing decoder")
	}
	if len(files.FontData) == 0 {
		t.Fatalf("pack font data empty")
	}
	if !strings.Contains(files.CSS, "ansi-link:hover") {
		t.Fatalf("pack css missing hover rule")
	}
}
