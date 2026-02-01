package html

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"

	_ "embed"
)

type TemplateData struct {
	Neotex      string
	FontDataURI template.URL
	CSS         template.CSS
	JS          template.JS
	OptionsBar  template.HTML
}

type PackFiles struct {
	HTML     string
	CSS      string
	JS       string
	FontName string
	FontData []byte
}

const optionsBarHTML = `<div id="options-bar">
  <label>Zoom <input id="zoom-range" type="range" min="7" max="64" step="7" value="14"></label>
  <label><input id="source-toggle" type="checkbox"> Source</label>
</div>`

//go:embed Web437_ToshibaSat_8x14.woff
var fontData []byte

const FontFileName = "Web437_ToshibaSat_8x14.woff"

func fontDataURI() string {
	return "data:font/woff;base64," + base64.StdEncoding.EncodeToString(fontData)
}

func fontBytes() []byte {
	return fontData
}

var pageTemplate = template.Must(template.New("splitans-html").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>splitans export</title>
  <style>
{{.CSS}}
  </style>
</head>
<body>
<pre id="neo-source">
{{.Neotex}}
</pre>
<pre id="ansi-output"></pre>
{{.OptionsBar}}
<script>
(() => {
{{.JS}}
})();
</script>
</body>
</html>`))

var packHTMLTemplate = template.Must(template.New("splitans-html-pack").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>splitans export</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
<pre id="neo-source">
{{.Neotex}}
</pre>
<pre id="ansi-output"></pre>
{{.OptionsBar}}
<script src="app.js"></script>
</body>
</html>`))

const baseCSS = `    :root { --bg: #000; --fg: #c0c0c0; }
    @font-face {
      font-family: 'Web437_ToshibaSat_8x14';
      src: url(__FONT_SRC__) format('woff');
      font-display: swap;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--fg);
      font-family: 'Web437_ToshibaSat_8x14', monospace;
      font-size: 28px;
      font-style: normal;
      line-height: 1.2;
      letter-spacing: 0;
      font-variant-ligatures: none;
      font-feature-settings: 'liga' 0;
      -webkit-font-smoothing: none;
      text-rendering: optimizeSpeed;
      padding: 16px;
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 12px;
    }
    #neo-source { display: none; }
    #ansi-output {
      white-space: pre;
      font-family: 'Web437_ToshibaSat_8x14', monospace;
      font-size: 28px;
      font-style: normal;
      line-height: 1.2;
      letter-spacing: 0;
      font-variant-ligatures: none;
      font-feature-settings: 'liga' 0;
      -webkit-font-smoothing: none;
      text-rendering: optimizeSpeed;
      background: #000;
      color: var(--fg);
      padding: 12px;
      overflow: auto;
    }
    a { color: inherit; }
    #options-bar {
      position: fixed;
      bottom: 10px;
      left: 50%;
      transform: translateX(-50%);
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 6px 10px;
      background: rgba(0, 0, 0, 0.8);
      color: #aaa;
      border: 1px solid #222;
      border-radius: 6px;
      font-size: 12px;
      font-family: 'Web437_ToshibaSat_8x14', monospace;
      pointer-events: auto;
    }
    #options-bar label { display: inline-flex; align-items: center; gap: 4px; }
    #options-bar input[type="range"] { accent-color: #666; }
    .ansi-cell { color: var(--fg, inherit); background: var(--bg, transparent); text-decoration: none; }
    .ansi-link { text-decoration: none; }
    .ansi-link:hover {
      color: var(--hover-fg, var(--fg, inherit));
      background: var(--hover-bg, #005400);
    }
    .ansi-italic { font-style: italic; }
    .ansi-underline { text-decoration: underline; }
    .ansi-blink { animation: ansi-blink 1s step-end infinite; }
    .ansi-dim { opacity: 0.8; }
    .ansi-hidden { opacity: 0; }
    @keyframes ansi-blink { 50% { opacity: 0.2; } }`

const baseJS = `  const palette = [
    [0x00, 0x00, 0x00], [0xAA, 0x00, 0x00], [0x00, 0xAA, 0x00], [0xAA, 0x55, 0x00],
    [0x00, 0x00, 0xAA], [0xAA, 0x00, 0xAA], [0x00, 0xAA, 0xAA], [0xAA, 0xAA, 0xAA],
    [0x55, 0x55, 0x55], [0xFF, 0x55, 0x55], [0x55, 0xFF, 0x55], [0xFF, 0xFF, 0x55],
    [0x55, 0x55, 0xFF], [0xFF, 0x55, 0xFF], [0x55, 0xFF, 0xFF], [0xFF, 0xFF, 0xFF]
  ];

  const defaultState = () => ({
    fg: { type: 'standard', index: 7 },
    bg: { type: 'standard', index: 0 },
    dim: false,
    italic: false,
    underline: false,
    blink: false,
    reverse: false,
    hidden: false,
  });

  const cloneState = (s) => ({
    fg: { ...s.fg },
    bg: { ...s.bg },
    dim: s.dim,
    italic: s.italic,
    underline: s.underline,
    blink: s.blink,
    reverse: s.reverse,
    hidden: s.hidden,
  });

  const fgMap = {
    Fk: 0, Fr: 1, Fg: 2, Fy: 3, Fb: 4, Fm: 5, Fc: 6, Fw: 7,
    FK: 8, FR: 9, FG: 10, FY: 11, FB: 12, FM: 13, FC: 14, FW: 15, FD: 7,
  };
  const bgMap = {
    Bk: 0, Br: 1, Bg: 2, By: 3, Bb: 4, Bm: 5, Bc: 6, Bw: 7,
    BK: 8, BR: 9, BG: 10, BY: 11, BB: 12, BM: 13, BC: 14, BW: 15, BD: 0,
  };

  const parseRGB = (code) => {
    const hex = code.slice(1);
    return [
      parseInt(hex.slice(0, 2), 16),
      parseInt(hex.slice(2, 4), 16),
      parseInt(hex.slice(4, 6), 16),
    ];
  };

  const indexedToRGB = (index) => {
    if (index < 16) return palette[index];
    if (index >= 232) {
      const level = 8 + (index - 232) * 10;
      return [level, level, level];
    }
    const i = index - 16;
    const r = Math.floor(i / 36);
    const g = Math.floor((i % 36) / 6);
    const b = i % 6;
    const conv = (v) => (v === 0 ? 0 : 55 + v * 40);
    return [conv(r), conv(g), conv(b)];
  };

  const colorToCSS = (color) => {
    if (!color) return null;
    if (color.type === 'standard') return palette[color.index % palette.length];
    if (color.type === 'rgb') return [color.r, color.g, color.b];
    if (color.type === 'indexed') return indexedToRGB(color.index);
    return null;
  };

  const rgbString = (rgb) => 'rgb(' + rgb[0] + ', ' + rgb[1] + ', ' + rgb[2] + ')';
  const toHex = (v) => v.toString(16).padStart(2, '0').toUpperCase();
  const rgbToHex = (rgb) => toHex(rgb[0]) + toHex(rgb[1]) + toHex(rgb[2]);

  const applyCode = (code, state) => {
    if (code === 'R0') {
      const next = defaultState();
      Object.assign(state, next);
      return;
    }
    if (fgMap[code] !== undefined) {
      state.fg = { type: 'standard', index: fgMap[code] };
      return;
    }
    if (bgMap[code] !== undefined) {
      state.bg = { type: 'standard', index: bgMap[code] };
      return;
    }
    if (code === 'EM') { state.dim = true; return; }
    if (code === 'Em') { state.dim = false; return; }
    if (code === 'EI') { state.italic = true; return; }
    if (code === 'Ei') { state.italic = false; return; }
    if (code === 'EU') { state.underline = true; return; }
    if (code === 'Eu') { state.underline = false; return; }
    if (code === 'EB') { state.blink = true; return; }
    if (code === 'Eb') { state.blink = false; return; }
    if (code === 'ER') { state.reverse = true; return; }
    if (code === 'Er') { state.reverse = false; return; }
    if (/^[FB][0-9A-Fa-f]{6}$/.test(code)) {
      const rgb = parseRGB(code);
      if (code[0] === 'F') state.fg = { type: 'rgb', r: rgb[0], g: rgb[1], b: rgb[2] };
      else state.bg = { type: 'rgb', r: rgb[0], g: rgb[1], b: rgb[2] };
      return;
    }
    if (/^[FB]\d{1,3}$/.test(code)) {
      const value = parseInt(code.slice(1), 10);
      if (Number.isNaN(value)) return;
      if (value < 0 || value > 255) return;
      if (code[0] === 'F') state.fg = { type: 'indexed', index: value };
      else state.bg = { type: 'indexed', index: value };
    }
  };

  const parseHoverColor = (code) => {
    // HF / HB standard (HFk, HFr...), RGB (HFRRGGBB/HBRRGGBB), indexed (HF123)
    if (!code || code.length < 2) return null;
    const kind = code.slice(0, 2);
    const body = code.slice(2);
    const isFg = kind === 'HF';
    const isBg = kind === 'HB';
    if (!isFg && !isBg) return null;

    const stdMap = {
      k: 0, r: 1, g: 2, y: 3, b: 4, m: 5, c: 6, w: 7,
      K: 8, R: 9, G: 10, Y: 11, B: 12, M: 13, C: 14, W: 15,
    };

    if (body.length === 1 && stdMap[body] !== undefined) {
      return { type: 'standard', index: stdMap[body], target: isFg ? 'fg' : 'bg' };
    }

    if (body.length === 6 && /^[0-9A-Fa-f]{6}$/.test(body)) {
      return {
        type: 'rgb',
        r: parseInt(body.slice(0, 2), 16),
        g: parseInt(body.slice(2, 4), 16),
        b: parseInt(body.slice(4, 6), 16),
        target: isFg ? 'fg' : 'bg',
      };
    }

    if (/^\d{1,3}$/.test(body)) {
      const value = parseInt(body, 10);
      if (value >= 0 && value <= 255) {
        return { type: 'indexed', index: value, target: isFg ? 'fg' : 'bg' };
      }
    }
    return null;
  };

  const parseSequences = (seqLine) => {
    const styles = [];
    if (!seqLine) return styles;
    for (const raw of seqLine.split(';')) {
      const entry = raw.trim();
      if (!entry || entry.startsWith('!')) continue;
      const parts = entry.split(':');
      if (parts.length < 2) continue;
      const position = parseInt(parts[0], 10) - 1;
      if (Number.isNaN(position)) continue;
      const codes = parts.slice(1).join(':').split(',').map((c) => c.trim()).filter(Boolean);
      const style = { position, codes: [], hyperlink: null, unlink: false, hoverFg: null, hoverBg: null };
      for (const code of codes) {
        if (code === 'Hl') { style.unlink = true; continue; }
        if (code.startsWith('HL:<') && code.endsWith('>')) {
          style.hyperlink = { url: code.slice(4, -1) };
          continue;
        }
        const hover = parseHoverColor(code);
        if (hover) {
          if (hover.target === 'fg') style.hoverFg = hover;
          else style.hoverBg = hover;
          continue;
        }
        style.codes.push(code);
      }
      if (style.codes.length || style.hyperlink || style.unlink || style.hoverFg || style.hoverBg) styles.push(style);
    }
    styles.sort((a, b) => a.position - b.position);
    return styles;
  };

  const extractWidth = (raw, normalized) => {
    const globalMatch = raw.match(/!TW(\d+)/);
    if (globalMatch) return parseInt(globalMatch[1], 10);
    for (let i = 0; i < normalized.length; i += 1) {
      const line = normalized[i];
      const idx = line.indexOf(' | ');
      if (idx < 0) continue;
      const seq = line.slice(idx + 3);
      const match = seq.match(/!TW(\d+)/);
      if (match) return parseInt(match[1], 10);
    }
    return null;
  };

  const parseNeotex = (raw) => {
    const lines = [];
    const normalized = raw.replace(/\r\n?/g, '\n').split('\n');
    const width = extractWidth(raw, normalized);
    let seenTW = false;

    for (let i = 0; i < normalized.length; i += 1) {
      const line = normalized[i];
      if (!line) continue;

      const sepIndex = line.lastIndexOf(' | ');
      if (sepIndex < 0) continue;
      const seqPart = line.slice(sepIndex + 3);
      if (!seenTW) {
        if (!seqPart.includes('!TW')) continue;
        seenTW = true;
      }

      if (width !== null) {
        const runes = Array.from(line);
        const sepStart = width;
        const sepEnd = width + 3;
        if (runes.length >= sepEnd) {
          const sep = runes.slice(sepStart, sepEnd).join('');
          if (sep === ' | ') {
            const text = runes.slice(0, width).join('');
            const seq = runes.slice(sepEnd).join('');
            lines.push({ text, seq });
            continue;
          }
        }
        if (sepIndex >= 0) {
          lines.push({ text: line.slice(0, sepIndex), seq: seqPart });
          continue;
        }
      }

      lines.push({ text: line.slice(0, sepIndex), seq: seqPart });
    }
    return lines;
  };

  let showSource = false;

  const renderLine = (text, seqLine, state, hyperlink, hoverFgState, hoverBgState) => {
    const styles = parseSequences(seqLine);
    const result = [];
    const runes = Array.from(text);
    let cursor = 0;
    let current = cloneState(state);
    let currentLink = hyperlink;
    let currentHoverFg = hoverFgState || null;
    let currentHoverBg = hoverBgState || null;

    const pushRun = (until) => {
      if (until <= cursor) return;
      const segment = runes.slice(cursor, until).join('');
      result.push({ text: segment, state: cloneState(current), hyperlink: currentLink, hoverFg: currentHoverFg, hoverBg: currentHoverBg });
      cursor = until;
    };

    for (const change of styles) {
      const pos = Math.min(change.position, runes.length);
      pushRun(pos);
      const nextState = cloneState(current);
      for (const code of change.codes) applyCode(code, nextState);
      current = nextState;

      if (change.hoverFg) currentHoverFg = change.hoverFg;
      if (change.hoverBg) currentHoverBg = change.hoverBg;

      if (change.unlink) currentLink = null;
      if (change.hyperlink) {
        currentLink = change.hyperlink;
        // store hover colors on the active link
        if (currentHoverFg) currentLink.hoverFg = currentHoverFg;
        if (currentHoverBg) currentLink.hoverBg = currentHoverBg;
      }
    }
    pushRun(runes.length);
    return { segments: result, state: current, hyperlink: currentLink, hoverFg: currentHoverFg, hoverBg: currentHoverBg };
  };

  const render = () => {
    const source = document.getElementById('neo-source');
    const target = document.getElementById('ansi-output');
    if (!source || !target) return;
    if (showSource) {
      target.textContent = source.textContent || '';
      return;
    }
    const lines = parseNeotex(source.textContent || '');
    let state = defaultState();
    let hyperlink = null;
    let hoverFg = null;
    let hoverBg = null;
    const frag = document.createDocumentFragment();

    for (let i = 0; i < lines.length; i += 1) {
      const { text, seq } = lines[i];
      const { segments, state: nextState, hyperlink: nextLink, hoverFg: nextHoverFg, hoverBg: nextHoverBg } = renderLine(text, seq, state, hyperlink, hoverFg, hoverBg);
      for (const part of segments) {
        const el = part.hyperlink ? document.createElement('a') : document.createElement('span');
        el.textContent = part.text;
        el.classList.add('ansi-cell');
        applyColorClasses(el, part.state.fg, false, true);
        applyColorClasses(el, part.state.bg, false, false);
        if (part.state.italic) el.classList.add('ansi-italic');
        if (part.state.underline) el.classList.add('ansi-underline');
        if (part.state.blink) el.classList.add('ansi-blink');
        if (part.state.dim) el.classList.add('ansi-dim');
        if (part.state.hidden) el.classList.add('ansi-hidden');
        if (part.hyperlink) {
          el.href = part.hyperlink.url;
          el.target = '_blank';
          el.rel = 'noopener noreferrer';
          el.classList.add('ansi-link');
          applyColorClasses(el, part.hoverFg, true, true);
          applyColorClasses(el, part.hoverBg, true, false);
        }
        frag.appendChild(el);
      }
      if (i < lines.length - 1) frag.appendChild(document.createTextNode('\n'));
      state = nextState;
      hyperlink = nextLink;
      hoverFg = nextHoverFg;
      hoverBg = nextHoverBg;
    }

    target.replaceChildren(frag);
  };

  const fgCodes = ['Fk','Fr','Fg','Fy','Fb','Fm','Fc','Fw','FK','FR','FG','FY','FB','FM','FC','FW'];
  const bgCodes = ['Bk','Br','Bg','By','Bb','Bm','Bc','Bw','BK','BR','BG','BY','BB','BM','BC','BW'];

  const dynamicRules = new Set();
  let dynamicStyleEl = null;

  const ensureDynamicStyle = () => {
    if (dynamicStyleEl) return dynamicStyleEl;
    dynamicStyleEl = document.createElement('style');
    dynamicStyleEl.id = 'dynamic-palette';
    document.head.appendChild(dynamicStyleEl);
    return dynamicStyleEl;
  };

  const addDynamicRule = (cls, prop, rgb) => {
    const key = cls + prop + rgb.join(',');
    if (dynamicRules.has(key)) return;
    dynamicRules.add(key);
    const styleEl = ensureDynamicStyle();
    styleEl.appendChild(document.createTextNode('.' + cls + ' { ' + prop + ': ' + rgbString(rgb) + '; }\n'));
  };

  const classForColor = (color, isHover, isFg) => {
    if (!color) return null;
    if (color.type === 'standard') {
      const codes = isFg ? fgCodes : bgCodes;
      const base = codes[color.index % codes.length];
      return isHover ? (isFg ? 'H' + base : 'H' + base) : base;
    }
    const rgb = colorToCSS(color) || [255, 255, 255];
    const hex = rgbToHex(rgb);
    const prefix = isHover ? (isFg ? 'HF_' : 'HB_') : (isFg ? 'F_' : 'B_');
    const cls = prefix + hex;
    const prop = isHover ? (isFg ? '--hover-fg' : '--hover-bg') : (isFg ? '--fg' : '--bg');
    addDynamicRule(cls, prop, rgb);
    return cls;
  };

  const applyColorClasses = (el, color, isHover, isFg) => {
    const cls = classForColor(color, isHover, isFg);
    if (cls) el.classList.add(cls);
  };

  const setupControls = () => {
    const target = document.getElementById('ansi-output');
    const zoom = document.getElementById('zoom-range');
    const sourceToggle = document.getElementById('source-toggle');

    const applyZoom = () => {
      const size = zoom ? Number(zoom.value) || 14 : 14;
      if (target) {
        target.style.fontSize = size + 'px';
        target.style.lineHeight = size + 'px';
      }
    };

    if (zoom) {
      zoom.addEventListener('input', applyZoom);
      applyZoom();
    }

    if (sourceToggle) {
      showSource = sourceToggle.checked;
      sourceToggle.addEventListener('change', () => {
        showSource = sourceToggle.checked;
        render();
      });
    }
  };

  setupControls();
  render();`

func buildStandardCSS() string {
	palette := [16][3]uint8{
		{0x00, 0x00, 0x00}, {0xAA, 0x00, 0x00}, {0x00, 0xAA, 0x00}, {0xAA, 0x55, 0x00},
		{0x00, 0x00, 0xAA}, {0xAA, 0x00, 0xAA}, {0x00, 0xAA, 0xAA}, {0xAA, 0xAA, 0xAA},
		{0x55, 0x55, 0x55}, {0xFF, 0x55, 0x55}, {0x55, 0xFF, 0x55}, {0xFF, 0xFF, 0x55},
		{0x55, 0x55, 0xFF}, {0xFF, 0x55, 0xFF}, {0x55, 0xFF, 0xFF}, {0xFF, 0xFF, 0xFF},
	}
	fgCodes := []string{"Fk", "Fr", "Fg", "Fy", "Fb", "Fm", "Fc", "Fw", "FK", "FR", "FG", "FY", "FB", "FM", "FC", "FW"}
	bgCodes := []string{"Bk", "Br", "Bg", "By", "Bb", "Bm", "Bc", "Bw", "BK", "BR", "BG", "BY", "BB", "BM", "BC", "BW"}

	var b strings.Builder
	for i, code := range fgCodes {
		r, g, bb := palette[i][0], palette[i][1], palette[i][2]
		fmt.Fprintf(&b, ".%s { --fg: rgb(%d,%d,%d); }\n", code, r, g, bb)
		fmt.Fprintf(&b, ".H%s { --hover-fg: rgb(%d,%d,%d); }\n", code, r, g, bb)
	}
	for i, code := range bgCodes {
		r, g, bb := palette[i][0], palette[i][1], palette[i][2]
		fmt.Fprintf(&b, ".%s { --bg: rgb(%d,%d,%d); }\n", code, r, g, bb)
		fmt.Fprintf(&b, ".H%s { --hover-bg: rgb(%d,%d,%d); }\n", code, r, g, bb)
	}
	return b.String()
}

// ExportHTML generates a standalone HTML document embedding neotex content and a JS decoder.
func ExportHTML(neotexContent string) (string, error) {
	paletteCSS := buildStandardCSS()
	css := strings.ReplaceAll(baseCSS, "__FONT_SRC__", fontDataURI()) + "\n" + paletteCSS
	js := baseJS

	data := TemplateData{
		Neotex:      neotexContent,
		FontDataURI: template.URL(fontDataURI()),
		CSS:         template.CSS(css),
		JS:          template.JS(js),
		OptionsBar:  template.HTML(optionsBarHTML),
	}

	var buf bytes.Buffer
	if err := pageTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render html export: %w", err)
	}

	return buf.String(), nil
}

// ExportHTMLPack generates separate HTML, CSS, JS and font bytes.
func ExportHTMLPack(neotexContent string) (PackFiles, error) {
	paletteCSS := buildStandardCSS()
	css := strings.ReplaceAll(baseCSS, "__FONT_SRC__", FontFileName) + "\n" + paletteCSS
	js := baseJS

	var htmlBuf bytes.Buffer
	if err := packHTMLTemplate.Execute(&htmlBuf, TemplateData{Neotex: neotexContent, OptionsBar: template.HTML(optionsBarHTML)}); err != nil {
		return PackFiles{}, fmt.Errorf("render html pack: %w", err)
	}

	return PackFiles{
		HTML:     htmlBuf.String(),
		CSS:      css,
		JS:       "(() => {\n" + js + "\n})();",
		FontName: FontFileName,
		FontData: fontBytes(),
	}, nil
}
