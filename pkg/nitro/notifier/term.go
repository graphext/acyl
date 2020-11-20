package notifier

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/pkg/errors"
)

// TerminalBackend is a notifier backend that writes to Output
type TerminalBackend struct {
	Output io.Writer
	Margin uint
}

var _ Backend = &TerminalBackend{}

const (
	resetCode  = "\u001b[0m"
	bwCode     = "\u001b[37;1m"
	redCode    = "\u001b[31m"
	yellowCode = "\u001b[33m"
	greenCode  = "\u001b[32m"
	whiteCode  = "\u001b[37m"
)

// emojis that "swallow" a space to the right in macOS
var insertSpaceEmojis = []string{"🛠", "❌", "☠️", "➡️", "⬇️", "↖️", "↙️", "↗️", "⬆️", "⬅️", "↘️"}

// emojis that take up an additional horizontal cell
var skipSpaceEmojis = []string{"🚦", "💣", "🏁", "☠️"}

// Send renders n to the terminal with a frame around it
func (tb *TerminalBackend) Send(n Notification) error {
	if tb.Output == nil {
		return errors.New("output is not set")
	}
	if tb.Margin == 0 {
		tb.Margin = 60
	}
	rn, err := n.Template.Render(n.Data)
	if err != nil {
		return errors.Wrap(err, "error rendering template")
	}
	rmargin := int(tb.Margin)

	brightwhite := func(s string) string { return bwCode + s + resetCode }
	vertical := []rune(brightwhite("║"))
	skipcount := func(s, skipstr string) int { return strings.Count(s, skipstr) * utf8.RuneCountInString(skipstr) }
	// padding assumes a string with a newline and without graphical bars
	// some emojis may screw this up, because they use variable amounts of horizontal space in TTYs that support them and we can't really detect how much
	padding := func(s string, f func(string) string) string {
		s = s[0 : len(s)-1] // strip trailing newline
		calcskip := func(rs []rune) int {
			skip := 0
			skip += skipcount(string(rs), resetCode)
			skip += skipcount(string(rs), bwCode)
			skip += skipcount(string(rs), whiteCode)
			skip += skipcount(string(rs), redCode)
			skip += skipcount(string(rs), greenCode)
			skip += skipcount(string(rs), yellowCode)
			for i := range rs {
				if unicode.Is(unicode.Variation_Selector, rs[i]) ||
					(runewidth.RuneWidth(rs[i]) == 2 && !unicode.Is(unicode.Variation_Selector, rs[i+1])) {
					skip++
				}
			}
			for _, s := range skipSpaceEmojis {
				skip -= strings.Count(string(rs), s)
			}
			return skip
		}
		b := &strings.Builder{}
		for _, l := range strings.Split(s, "\n") { // split removes all internal newlines
			// insert extra space after certain emojis to account for the space
			// which the terminal visually "swallows" in macOS
			if runtime.GOOS == "darwin" {
				rs := []rune(l)
				for i := range rs {
					for j := range insertSpaceEmojis {
						if rs[i] == []rune(insertSpaceEmojis[j])[0] {
							pad := 2
							if unicode.Is(unicode.Variation_Selector, rs[i+1]) {
								pad++
							}
							rs = append(rs, 0)
							copy(rs[i+pad:], rs[i+(pad-1):])
							rs[i+(pad-1)] = ' '
						}
					}
				}
				l = string(rs)
			}
			// skip counting all non-printing ANSI control runes
			rs := []rune(l)
			i := runewidth.StringWidth(l) - calcskip(rs)
			rout := []rune{}
			switch {
			case i < rmargin-2:
				// if a line is too short, pad it with spaces
				rout = append(rout, vertical...)
				rout = append(rout, []rune(f(l))...)
				rout = append(rout, []rune(strings.Repeat(" ", rmargin-(i+2)))...)
				rout = append(rout, vertical...)
			case i > rmargin-2:
				// if a line is too long, character wrap
				n := 0
				cl := make([]rune, rmargin+2) // current line buffer
				for j := range rs {
					cl[n] = rs[j]
					if (n - calcskip(cl)) == (rmargin - 3) {
						rout = append(rout, vertical...)
						rout = append(rout, []rune(f(string(cl)))...)
						rout = append(rout, vertical...)
						rout = append(rout, '\n')
						cl = make([]rune, rmargin+2) // reset line buffer
						n = 0                        // reset current line index
						continue
					}
					n++ // current line index
				}
				if n > 0 {
					rout = append(rout, vertical...)
					rout = append(rout, []rune(f(string(cl)))...)
					if (n - calcskip(cl)) < rmargin-2 {
						rout = append(rout, []rune(strings.Repeat(" ", rmargin-(n+2)))...)
					}
					rout = append(rout, vertical...)
				}
			case i == rmargin-2:
				rout = append(rout, vertical...)
				rout = append(rout, []rune(f(string(rs)))...)
				rout = append(rout, vertical...)
			}
			b.Write([]byte(string(rout)))
			b.Write([]byte("\n"))
		}
		return b.String()
	}
	fmt.Fprintf(tb.Output, "\n")
	fmt.Fprintf(tb.Output, brightwhite("╔"+strings.Repeat("═", rmargin-2)+"╗\n"))
	fmt.Fprintf(tb.Output, padding(rn.Title+"\n\n", brightwhite))
	for _, s := range rn.Sections {
		var f func(string) string
		switch s.Style {
		case "good":
			f = func(s string) string { return greenCode + s + resetCode }
		case "warning":
			f = func(s string) string { return yellowCode + s + resetCode }
		case "danger":
			f = func(s string) string { return redCode + s + resetCode }
		default:
			f = func(s string) string { return whiteCode + s + resetCode }
		}
		fmt.Fprintf(tb.Output, padding(s.Title+"\n", brightwhite))
		fmt.Fprintf(tb.Output, padding(s.Text+"\n", f))
	}
	fmt.Fprintf(tb.Output, brightwhite("╚"+strings.Repeat("═", rmargin-2)+"╝\n"))
	fmt.Fprintf(tb.Output, "\n")
	return nil
}
