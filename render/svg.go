package render

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/h4nxxlmao/qr2/qr"
)

type paintGroup struct {
	color string
	d     strings.Builder
}

type svgBuilder struct {
	spec   Spec
	groups []*paintGroup
	index  map[string]int
	logo   logoGeom
}

type logoGeom struct {
	enabled bool
	dataURI string
	cx      float64
	cy      float64
	size    float64
	plate   float64
	shape   string
	clear   bool
}

func logoGeometry(spec Spec, total float64) logoGeom {
	if spec.LogoDataURI == "" {
		return logoGeom{}
	}
	size := total * spec.LogoSizePct
	margin := size * spec.LogoMarginPct
	return logoGeom{
		enabled: true,
		dataURI: spec.LogoDataURI,
		cx:      total / 2,
		cy:      total / 2,
		size:    size,
		plate:   size + 2*margin,
		shape:   spec.LogoShape,
		clear:   spec.LogoClear,
	}
}

func (l logoGeom) skip(x, y int) bool {
	if !l.enabled || !l.clear {
		return false
	}
	dx := float64(x) + 0.5 - l.cx
	dy := float64(y) + 0.5 - l.cy
	half := l.plate / 2
	if l.shape == "circle" {
		return dx*dx+dy*dy <= half*half
	}
	return absFloat(dx) <= half && absFloat(dy) <= half
}

func (l logoGeom) drawDefs(b *strings.Builder) {
	if !l.enabled || l.shape == "square" {
		return
	}
	b.WriteString(`<defs><clipPath id="qrlogoClip">`)
	x := l.cx - l.plate/2
	y := l.cy - l.plate/2
	if l.shape == "circle" {
		b.WriteString(`<circle cx="`)
		b.WriteString(num(l.cx))
		b.WriteString(`" cy="`)
		b.WriteString(num(l.cy))
		b.WriteString(`" r="`)
		b.WriteString(num(l.plate / 2))
		b.WriteString(`"/>`)
	} else {
		b.WriteString(`<rect x="`)
		b.WriteString(num(x))
		b.WriteString(`" y="`)
		b.WriteString(num(y))
		b.WriteString(`" width="`)
		b.WriteString(num(l.plate))
		b.WriteString(`" height="`)
		b.WriteString(num(l.plate))
		b.WriteString(`" rx="`)
		b.WriteString(num(l.plate * 0.18))
		b.WriteString(`"/>`)
	}
	b.WriteString(`</clipPath></defs>`)
}

func (l logoGeom) draw(b *strings.Builder, spec Spec) {
	if !l.enabled {
		return
	}
	fill := spec.Background
	if spec.Transparent || fill == "" || fill == "transparent" || fill == "none" {
		fill = "#ffffff"
	}
	x := l.cx - l.plate/2
	y := l.cy - l.plate/2
	b.WriteString(`<g class="logo">`)
	switch l.shape {
	case "circle":
		b.WriteString(`<circle cx="`)
		b.WriteString(num(l.cx))
		b.WriteString(`" cy="`)
		b.WriteString(num(l.cy))
		b.WriteString(`" r="`)
		b.WriteString(num(l.plate / 2))
		b.WriteString(`" fill="`)
		b.WriteString(fill)
		b.WriteString(`"/>`)
	case "rounded":
		b.WriteString(`<rect x="`)
		b.WriteString(num(x))
		b.WriteString(`" y="`)
		b.WriteString(num(y))
		b.WriteString(`" width="`)
		b.WriteString(num(l.plate))
		b.WriteString(`" height="`)
		b.WriteString(num(l.plate))
		b.WriteString(`" rx="`)
		b.WriteString(num(l.plate * 0.18))
		b.WriteString(`" fill="`)
		b.WriteString(fill)
		b.WriteString(`"/>`)
	default:
		b.WriteString(`<rect x="`)
		b.WriteString(num(x))
		b.WriteString(`" y="`)
		b.WriteString(num(y))
		b.WriteString(`" width="`)
		b.WriteString(num(l.plate))
		b.WriteString(`" height="`)
		b.WriteString(num(l.plate))
		b.WriteString(`" fill="`)
		b.WriteString(fill)
		b.WriteString(`"/>`)
	}
	b.WriteString(`<image href="`)
	b.WriteString(l.dataURI)
	b.WriteString(`" x="`)
	b.WriteString(num(l.cx - l.size/2))
	b.WriteString(`" y="`)
	b.WriteString(num(l.cy - l.size/2))
	b.WriteString(`" width="`)
	b.WriteString(num(l.size))
	b.WriteString(`" height="`)
	b.WriteString(num(l.size))
	b.WriteString(`" preserveAspectRatio="xMidYMid meet"`)
	if l.shape != "square" {
		b.WriteString(` clip-path="url(#qrlogoClip)"`)
	}
	b.WriteString(`/></g>`)
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// RenderSVG returns a complete SVG document for the symbol.
func RenderSVG(sym *qr.Symbol, spec Spec) ([]byte, error) {
	if sym == nil || sym.Width <= 0 || sym.Height <= 0 || len(sym.Modules) != sym.Width*sym.Height {
		return nil, errors.New("render: invalid symbol")
	}
	spec = normalize(spec)

	quiet := spec.QuietZone
	total := sym.Width + quiet*2
	size := spec.Size
	if size <= 0 {
		size = total
	}
	if size > 16384 {
		size = 16384
	}

	b := &svgBuilder{
		spec:  spec,
		index: make(map[string]int),
		logo:  logoGeometry(spec, float64(total)),
	}
	b.drawModules(sym, quiet)

	var sb strings.Builder
	sb.Grow(256 + len(b.groups)*128 + total*total/8)
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 `)
	sb.WriteString(strconv.Itoa(total))
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(total))
	sb.WriteString(`" width="`)
	sb.WriteString(strconv.Itoa(size))
	sb.WriteString(`" height="`)
	sb.WriteString(strconv.Itoa(size))
	sb.WriteString(`" role="img"`)
	if spec.Shape == "square" && spec.ModuleScale >= 0.999 && spec.CornerRadius == 0 {
		sb.WriteString(` shape-rendering="crispEdges"`)
	}
	sb.WriteString(`>`)
	sb.WriteString(`<title>QR code</title>`)
	b.logo.drawDefs(&sb)
	if !spec.Transparent {
		sb.WriteString(`<rect width="`)
		sb.WriteString(strconv.Itoa(total))
		sb.WriteString(`" height="`)
		sb.WriteString(strconv.Itoa(total))
		sb.WriteString(`" fill="`)
		sb.WriteString(spec.Background)
		sb.WriteString(`"/>`)
	}

	transform := transformFor(spec, float64(total))
	if transform != "" {
		sb.WriteString(`<g transform="`)
		sb.WriteString(transform)
		sb.WriteString(`">`)
	}
	for _, g := range b.groups {
		if g.d.Len() == 0 {
			continue
		}
		sb.WriteString(`<path d="`)
		sb.WriteString(g.d.String())
		sb.WriteString(`" fill="`)
		sb.WriteString(g.color)
		sb.WriteString(`"/>`)
	}
	b.logo.draw(&sb, spec)
	if transform != "" {
		sb.WriteString(`</g>`)
	}
	sb.WriteString(`</svg>`)
	return []byte(sb.String()), nil
}

func (b *svgBuilder) groupFor(color string) *strings.Builder {
	if i, ok := b.index[color]; ok {
		return &b.groups[i].d
	}
	g := &paintGroup{color: color}
	b.groups = append(b.groups, g)
	b.index[color] = len(b.groups) - 1
	return &g.d
}

func (b *svgBuilder) colorFor(role qr.Role) string {
	switch role {
	case qr.RoleFinderOuter:
		if b.spec.FinderOuter != "" {
			return b.spec.FinderOuter
		}
	case qr.RoleFinderInner:
		if b.spec.FinderInner != "" {
			return b.spec.FinderInner
		}
	case qr.RoleTiming:
		if b.spec.Timing != "" {
			return b.spec.Timing
		}
	case qr.RoleAlignment:
		if b.spec.Alignment != "" {
			return b.spec.Alignment
		}
	case qr.RoleFormat:
		if b.spec.FormatColor != "" {
			return b.spec.FormatColor
		}
	case qr.RoleVersion:
		if b.spec.VersionColor != "" {
			return b.spec.VersionColor
		}
	}
	return b.spec.Foreground
}

func (b *svgBuilder) drawModules(sym *qr.Symbol, quiet int) {
	shape := b.spec.Shape
	if shape == "square" && b.spec.ModuleScale >= 0.999 && b.spec.CornerRadius == 0 && !(b.logo.enabled && b.logo.clear) {
		b.drawSquareRuns(sym, quiet)
		return
	}
	for y := 0; y < sym.Height; y++ {
		for x := 0; x < sym.Width; x++ {
			if sym.Module(x, y) == 0 {
				continue
			}
			role := sym.RoleAt(x, y)
			if role == qr.RoleFinderWhite {
				continue
			}
			if b.logo.skip(x+quiet, y+quiet) {
				continue
			}
			d := b.groupFor(b.colorFor(role))
			mx := float64(quiet+x) + (1-b.spec.ModuleScale)/2
			my := float64(quiet+y) + (1-b.spec.ModuleScale)/2
			switch shape {
			case "dots":
				appendCircle(d, mx+0.5, my+0.5, b.spec.ModuleScale/2)
			case "diamond":
				appendDiamond(d, mx+0.5, my+0.5, b.spec.ModuleScale/2)
			case "rounded":
				appendRoundedRect(d, mx, my, b.spec.ModuleScale, b.spec.CornerRadius*b.spec.ModuleScale)
			default:
				appendRect(d, mx, my, b.spec.ModuleScale)
			}
		}
	}
}

func (b *svgBuilder) drawSquareRuns(sym *qr.Symbol, quiet int) {
	for y := 0; y < sym.Height; y++ {
		x := 0
		for x < sym.Width {
			if sym.Module(x, y) == 0 {
				x++
				continue
			}
			role := sym.RoleAt(x, y)
			if role == qr.RoleFinderWhite {
				x++
				continue
			}
			color := b.colorFor(role)
			start := x
			for x < sym.Width && sym.Module(x, y) != 0 && sym.RoleAt(x, y) != qr.RoleFinderWhite && b.colorFor(sym.RoleAt(x, y)) == color {
				x++
			}
			d := b.groupFor(color)
			appendRun(d, quiet+start, quiet+y, x-start)
		}
	}
}

func appendRun(b *strings.Builder, x, y, n int) {
	b.WriteByte('M')
	b.WriteString(strconv.Itoa(x))
	b.WriteByte(' ')
	b.WriteString(strconv.Itoa(y))
	b.WriteString("h")
	b.WriteString(strconv.Itoa(n))
	b.WriteString("v1h-")
	b.WriteString(strconv.Itoa(n))
	b.WriteByte('z')
}

func appendRect(b *strings.Builder, x, y, s float64) {
	b.WriteByte('M')
	b.WriteString(num(x))
	b.WriteByte(' ')
	b.WriteString(num(y))
	b.WriteString("h")
	b.WriteString(num(s))
	b.WriteString("v")
	b.WriteString(num(s))
	b.WriteString("h-")
	b.WriteString(num(s))
	b.WriteByte('z')
}

func appendRoundedRect(b *strings.Builder, x, y, s, r float64) {
	if r <= 0.0001 {
		appendRect(b, x, y, s)
		return
	}
	if r > s/2 {
		r = s / 2
	}
	b.WriteByte('M')
	b.WriteString(num(x + r))
	b.WriteByte(' ')
	b.WriteString(num(y))
	b.WriteString("H")
	b.WriteString(num(x + s - r))
	b.WriteString("Q")
	b.WriteString(num(x + s))
	b.WriteByte(' ')
	b.WriteString(num(y))
	b.WriteByte(' ')
	b.WriteString(num(x + s))
	b.WriteByte(' ')
	b.WriteString(num(y + r))
	b.WriteString("V")
	b.WriteString(num(y + s - r))
	b.WriteString("Q")
	b.WriteString(num(x + s))
	b.WriteByte(' ')
	b.WriteString(num(y + s))
	b.WriteByte(' ')
	b.WriteString(num(x + s - r))
	b.WriteByte(' ')
	b.WriteString(num(y + s))
	b.WriteString("H")
	b.WriteString(num(x + r))
	b.WriteString("Q")
	b.WriteString(num(x))
	b.WriteByte(' ')
	b.WriteString(num(y + s))
	b.WriteByte(' ')
	b.WriteString(num(x))
	b.WriteByte(' ')
	b.WriteString(num(y + s - r))
	b.WriteString("V")
	b.WriteString(num(y + r))
	b.WriteString("Q")
	b.WriteString(num(x))
	b.WriteByte(' ')
	b.WriteString(num(y))
	b.WriteByte(' ')
	b.WriteString(num(x + r))
	b.WriteByte(' ')
	b.WriteString(num(y))
	b.WriteByte('z')
}

func appendCircle(b *strings.Builder, cx, cy, r float64) {
	b.WriteByte('M')
	b.WriteString(num(cx - r))
	b.WriteByte(' ')
	b.WriteString(num(cy))
	b.WriteString("A")
	b.WriteString(num(r))
	b.WriteByte(' ')
	b.WriteString(num(r))
	b.WriteString(" 0 1 0 ")
	b.WriteString(num(cx + r))
	b.WriteByte(' ')
	b.WriteString(num(cy))
	b.WriteString("A")
	b.WriteString(num(r))
	b.WriteByte(' ')
	b.WriteString(num(r))
	b.WriteString(" 0 1 0 ")
	b.WriteString(num(cx - r))
	b.WriteByte(' ')
	b.WriteString(num(cy))
	b.WriteByte('z')
}

func appendDiamond(b *strings.Builder, cx, cy, r float64) {
	b.WriteByte('M')
	b.WriteString(num(cx))
	b.WriteByte(' ')
	b.WriteString(num(cy - r))
	b.WriteString("L")
	b.WriteString(num(cx + r))
	b.WriteByte(' ')
	b.WriteString(num(cy))
	b.WriteString("L")
	b.WriteString(num(cx))
	b.WriteByte(' ')
	b.WriteString(num(cy + r))
	b.WriteString("L")
	b.WriteString(num(cx - r))
	b.WriteByte(' ')
	b.WriteString(num(cy))
	b.WriteByte('z')
}

func num(v float64) string {
	return strconv.FormatFloat(math.Round(v*1000)/1000, 'f', -1, 64)
}

func transformFor(spec Spec, total float64) string {
	var parts []string
	if spec.Mirror {
		parts = append(parts, "translate("+num(total)+" 0) scale(-1 1)")
	}
	if spec.Rotation != 0 {
		parts = append(parts, "rotate("+num(spec.Rotation)+" "+num(total/2)+" "+num(total/2)+")")
	}
	return strings.Join(parts, " ")
}

func normalize(spec Spec) Spec {
	def := Defaults()
	if spec.Size < 0 {
		spec.Size = 0
	}
	if spec.Size > 16384 {
		spec.Size = 16384
	}
	if spec.QuietZone < 0 {
		spec.QuietZone = 0
	}
	if spec.QuietZone > 16 {
		spec.QuietZone = 16
	}
	if spec.QuietZone == 0 && !spec.Transparent {
		// A zero quiet zone is allowed; the UI warns. Do not silently add one.
	}
	if spec.Foreground == "" {
		spec.Foreground = def.Foreground
	}
	if spec.Background == "" {
		spec.Background = def.Background
	}
	spec.Foreground = safeColor(spec.Foreground, def.Foreground)
	spec.Background = safeColor(spec.Background, def.Background)
	spec.FinderOuter = safeColor(spec.FinderOuter, "")
	spec.FinderInner = safeColor(spec.FinderInner, "")
	spec.Timing = safeColor(spec.Timing, "")
	spec.Alignment = safeColor(spec.Alignment, "")
	spec.FormatColor = safeColor(spec.FormatColor, "")
	spec.VersionColor = safeColor(spec.VersionColor, "")
	if spec.Shape == "" {
		spec.Shape = def.Shape
	}
	switch spec.Shape {
	case "square", "rounded", "dots", "diamond":
	default:
		spec.Shape = "square"
	}
	if math.IsNaN(spec.ModuleScale) || math.IsInf(spec.ModuleScale, 0) {
		spec.ModuleScale = 1
	}
	if spec.ModuleScale < 0.5 {
		spec.ModuleScale = 0.5
	}
	if spec.ModuleScale > 1 {
		spec.ModuleScale = 1
	}
	if math.IsNaN(spec.CornerRadius) || math.IsInf(spec.CornerRadius, 0) {
		spec.CornerRadius = 0
	}
	if spec.CornerRadius < 0 {
		spec.CornerRadius = 0
	}
	if spec.CornerRadius > 0.5 {
		spec.CornerRadius = 0.5
	}
	if math.IsNaN(spec.Rotation) || math.IsInf(spec.Rotation, 0) {
		spec.Rotation = 0
	}
	spec.Rotation = math.Mod(spec.Rotation, 360)
	if spec.Rotation < 0 {
		spec.Rotation += 360
	}
	spec.LogoDataURI = safeLogoDataURI(spec.LogoDataURI)
	if spec.LogoDataURI == "" {
		spec.LogoSizePct = 0
		spec.LogoMarginPct = 0
		return spec
	}
	if math.IsNaN(spec.LogoSizePct) || spec.LogoSizePct <= 0 {
		spec.LogoSizePct = 0.18
	}
	if spec.LogoSizePct < 0.05 {
		spec.LogoSizePct = 0.05
	}
	if spec.LogoSizePct > 0.45 {
		spec.LogoSizePct = 0.45
	}
	if math.IsNaN(spec.LogoMarginPct) || spec.LogoMarginPct < 0 {
		spec.LogoMarginPct = 0
	}
	if spec.LogoMarginPct > 0.5 {
		spec.LogoMarginPct = 0.5
	}
	switch spec.LogoShape {
	case "square", "rounded", "circle":
	default:
		spec.LogoShape = "square"
	}
	return spec
}

func safeLogoDataURI(v string) string {
	const prefix = "data:image/"
	const marker = ";base64,"
	if !strings.HasPrefix(v, prefix) {
		return ""
	}
	index := strings.Index(v, marker)
	if index < 0 {
		return ""
	}
	mime := v[len(prefix):index]
	switch mime {
	case "png", "jpeg", "jpg", "webp":
	default:
		return ""
	}
	payload := v[index+len(marker):]
	if payload == "" || len(payload) > 8<<20 {
		return ""
	}
	for i := 0; i < len(payload); i++ {
		c := payload[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' {
			continue
		}
		return ""
	}
	return v
}

func safeColor(v, fallback string) string {
	if v == "" {
		return fallback
	}
	if v == "none" || v == "transparent" {
		return v
	}
	if v[0] != '#' {
		return fallback
	}
	n := len(v) - 1
	if n != 3 && n != 4 && n != 6 && n != 8 {
		return fallback
	}
	for i := 1; i < len(v); i++ {
		c := v[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fallback
		}
	}
	return v
}
