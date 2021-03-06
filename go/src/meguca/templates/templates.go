//go:generate qtc
//go:generate go-bindata -o bin_data.go --pkg templates --nometadata --prefix ../../../../mustache-pp ../../../../mustache-pp/...

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"strings"

	"meguca/auth"
	"meguca/config"
	"meguca/lang"

	"github.com/hoisie/mustache"
)

var (
	mustacheTemplates = map[string]*mustache.Template{}
)

func Page(ss *auth.Session, title string, html string) []byte {
	var buf bytes.Buffer
	writerenderPage(&buf, ss, title, html)
	return buf.Bytes()
}

func Board(
	b string,
	page, total int,
	ss *auth.Session,
	catalog bool,
	threadHTML []byte,
) []byte {
	boardConf := config.GetBoardConfig(b)
	title := boardConf.Title
	if b == "all" {
		title = lang.GT("aggregator")
	}
	html := renderBoard(
		threadHTML,
		b, title,
		page, total,
		catalog,
	)
	return Page(ss, title, html)
}

func Thread(
	id uint64,
	board, title string,
	abbrev bool,
	ss *auth.Session,
	postHTML []byte,
) []byte {
	html := renderThread(postHTML, id, board, title)
	return Page(ss, title, html)
}

func Landing(ss *auth.Session) []byte {
	title := lang.GT("main")
	html := renderLanding()
	return Page(ss, title, html)
}

func Stickers(ss *auth.Session, stickHTML []byte) []byte {
	html := renderStickers(stickHTML)
	title := lang.GT("stickers")
	return Page(ss, title, html)
}

func Admin(
	ss *auth.Session,
	cs config.BoardConfigs,
	staff auth.Staff,
	bans auth.BanRecords,
	log auth.ModLogRecords,
) []byte {
	html := renderAdmin(cs, staff, bans, log)
	title := lang.GT("Admin")
	return Page(ss, title, html)
}

func CompileMustache() (err error) {
	for _, name := range AssetNames() {
		if strings.HasSuffix(name, ".mustache") {
			tmpl, err := mustache.ParseString(string(MustAsset(name)))
			if err != nil {
				return fmt.Errorf("failed to compile %s: %v", name, err)
			}
			tmplName := name[:len(name)-9]
			mustacheTemplates[tmplName] = tmpl
		}
	}
	return
}

func renderMustache(name string, ctx interface{}) string {
	if tmpl, ok := mustacheTemplates[name]; ok {
		return tmpl.Render(ctx)
	}
	return ""
}
