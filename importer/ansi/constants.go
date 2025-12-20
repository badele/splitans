package ansi

// C1 control codes (7-bit representation)
var C1Sequences = map[string]string{
	"D":  "IND", // Index
	"E":  "NEL", // Next Line
	"H":  "HTS", // Horizontal Tab Set
	"M":  "RI",  // Reverse Index
	"P":  "DCS", // Device Control String
	"[":  "CSI", // Control Sequence Introducer
	"\\": "ST",  // String Terminator
	"]":  "OSC", // Operating System Command
}

// SGR codes descriptions
var SGRCodes = map[int]string{
	0:   "Reset",
	1:   "Bold",
	2:   "Dim",
	3:   "Italic",
	4:   "Underline",
	5:   "Blink",
	6:   "RapidBlink",
	7:   "Inverse",
	8:   "Invisible",
	9:   "StrikeThrough",
	21:  "DoubleUnderline",
	22:  "NormalIntensity",
	23:  "ItalicOff",
	24:  "UnderlineOff",
	25:  "BlinkOff",
	27:  "InverseOff",
	28:  "InvisibleOff",
	29:  "StrikeThroughOff",
	30:  "ForegroundBlack",
	31:  "ForegroundRed",
	32:  "ForegroundGreen",
	33:  "ForegroundYellow",
	34:  "ForegroundBlue",
	35:  "ForegroundMagenta",
	36:  "ForegroundCyan",
	37:  "ForegroundWhite",
	39:  "ForegroundDefault",
	40:  "BackgroundBlack",
	41:  "BackgroundRed",
	42:  "BackgroundGreen",
	43:  "BackgroundYellow",
	44:  "BackgroundBlue",
	45:  "BackgroundMagenta",
	46:  "BackgroundCyan",
	47:  "BackgroundWhite",
	49:  "BackgroundDefault",
	53:  "OverlineOn",
	55:  "OverlineOff",
	59:  "UnderlineColorDefault",
	90:  "ForegroundBrightBlack",
	91:  "ForegroundBrightRed",
	92:  "ForegroundBrightGreen",
	93:  "ForegroundBrightYellow",
	94:  "ForegroundBrightBlue",
	95:  "ForegroundBrightMagenta",
	96:  "ForegroundBrightCyan",
	97:  "ForegroundBrightWhite",
	100: "BackgroundBrightBlack",
	101: "BackgroundBrightRed",
	102: "BackgroundBrightGreen",
	103: "BackgroundBrightYellow",
	104: "BackgroundBrightBlue",
	105: "BackgroundBrightMagenta",
	106: "BackgroundBrightCyan",
	107: "BackgroundBrightWhite",
}

// ED codes descriptions
var EDCodes = map[int]string{
	0: "EraseBelow",
	1: "EraseAbove",
	2: "EraseAll",
}
