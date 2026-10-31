package render

// Spec controls the vector output. Zero values are safe defaults.
type Spec struct {
	Size         int     `json:"size"`         // output width/height in px; 0 = one unit per module
	QuietZone    int     `json:"quietZone"`    // modules; default 4
	Foreground   string  `json:"foreground"`   // #rrggbb or #rrggbbaa
	Background   string  `json:"background"`   // #rrggbb or #rrggbbaa
	Transparent  bool    `json:"transparent"`  // omit the background rectangle
	Shape        string  `json:"shape"`        // square, rounded, dots, diamond
	ModuleScale  float64 `json:"moduleScale"`  // 0.5..1
	CornerRadius float64 `json:"cornerRadius"` // 0..0.5, as a fraction of the module
	FinderOuter  string  `json:"finderOuter"`
	FinderInner  string  `json:"finderInner"`
	Timing       string  `json:"timing"`
	Alignment    string  `json:"alignment"`
	FormatColor  string  `json:"formatColor"`
	VersionColor string  `json:"versionColor"`
	Rotation     float64 `json:"rotation"` // degrees
	Mirror       bool    `json:"mirror"`

	LogoDataURI   string  `json:"logoDataUri"`
	LogoSizePct   float64 `json:"logoSizePct"`   // fraction of the full design width, 0.05..0.45
	LogoMarginPct float64 `json:"logoMarginPct"` // fraction of the logo size
	LogoShape     string  `json:"logoShape"`     // square, rounded, circle
	LogoClear     bool    `json:"logoClear"`
}

// Defaults returns the baseline style used when the UI has no opinion.
func Defaults() Spec {
	return Spec{
		Size:         512,
		QuietZone:    4,
		Foreground:   "#111111",
		Background:   "#ffffff",
		Shape:        "square",
		ModuleScale:  1,
		CornerRadius: 0,
	}
}
