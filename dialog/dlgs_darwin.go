package dialog

import (
	"path/filepath"
	"strings"

	"github.com/sqweek/dialog/cocoa"
)

func (b *MsgBuilder) yesNo() bool {
	return cocoa.YesNoDlg(b.Msg, b.Dlg.Title)
}

func (b *MsgBuilder) info() {
	cocoa.InfoDlg(b.Msg, b.Dlg.Title)
}

func (b *MsgBuilder) error() {
	cocoa.ErrorDlg(b.Msg, b.Dlg.Title)
}

func (b *FileBuilder) load() (string, error) {
	return b.run(false)
}

func (b *FileBuilder) save() (string, error) {
	return b.run(true)
}

func (b *FileBuilder) saveWithFilter() (string, int, error) {
	f, ferr := b.run(true)
	if ferr != nil {
		return "", -1, ferr
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(f), "."))
	for i, filt := range b.Filters {
		for _, e := range filt.Extensions {
			if e == ext {
				return f, i, nil
			}
		}
	}
	return f, -1, nil
}

func (b *FileBuilder) run(save bool) (string, error) {
	star := false
	var exts []string
	for _, filt := range b.Filters {
		for _, ext := range filt.Extensions {
			if ext == "*" {
				star = true
			} else {
				exts = append(exts, ext)
			}
		}
	}
	if star && save {
		/* OSX doesn't allow the user to switch visible file types/extensions. Also
		** NSSavePanel's allowsOtherFileTypes property has no effect for an open
		** dialog, so if "*" is a possible extension we must always show all files. */
		exts = nil
	}
	f, err := cocoa.FileDlg(save, b.Dlg.Title, exts, star, b.StartDir, b.StartFile)
	if f == "" && err == nil {
		return "", ErrCancelled
	}
	return f, err
}

func (b *DirectoryBuilder) browse() (string, error) {
	f, err := cocoa.DirDlg(b.Dlg.Title, b.StartDir)
	if f == "" && err == nil {
		return "", ErrCancelled
	}
	return f, err
}
