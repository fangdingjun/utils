package utils

import (
	"net/http"

	"golang.org/x/text/message"
)

// GetI18nPrinter return a message.printer according the request header
// user can override default Accept-Language header by add the language header
func GetI18nPrinter(r *http.Request) *message.Printer {
	// i18n support
	lang := r.Header.Get("language")
	lang2 := r.Header.Get("Accept-Language")
	fallback := "en"
	tag := message.MatchLanguage(lang, lang2, fallback)
	return message.NewPrinter(tag)
}
