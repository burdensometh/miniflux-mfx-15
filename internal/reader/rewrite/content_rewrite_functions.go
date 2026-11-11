// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite // import "miniflux.app/v2/internal/reader/rewrite"

import (
	"encoding/base64"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"miniflux.app/v2/internal/model"
)

var textLinkRegex = regexp.MustCompile(`(^|[^"'])(https?://[^"\s]+)`)

// applyToSelection runs fn over every node matching selector and returns the
// serialized result of those nodes.
func applyToSelection(entryContent, selector string, fn func(*goquery.Selection)) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	selection := doc.Find(selector)
	selection.Each(func(_ int, s *goquery.Selection) {
		fn(s)
	})

	output, _ := selection.Html()
	return output
}

func titlelize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func addImageTitle(entryContent string) string {
	return applyToSelection(entryContent, "img", func(s *goquery.Selection) {
		alt, _ := s.Attr("alt")
		if alt != "" {
			s.SetAttr("title", titlelize(alt))
		}
	})
}

func addMailtoSubject(entryContent string) string {
	return applyToSelection(entryContent, `a[href^="mailto:"]`, func(s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !strings.Contains(href, "subject=") {
			s.SetAttr("href", href+"?subject="+url.QueryEscape("Subject"))
		}
	})
}

func addDynamicImage(entryContent string) string {
	return applyToSelection(entryContent, "img", func(s *goquery.Selection) {
		if src, _ := s.Attr("src"); src != "" {
			s.SetAttr("src", "proxy/"+src)
		}
		if srcset, _ := s.Attr("srcset"); srcset != "" {
			s.SetAttr("srcset", "proxy/"+srcset)
		}
	})
}

func addDynamicIframe(entryContent string) string {
	return applyToSelection(entryContent, "iframe", func(s *goquery.Selection) {
		if src, _ := s.Attr("src"); src != "" {
			s.SetAttr("src", "proxy/"+src)
		}
	})
}

func fixMediumImages(entryContent string) string {
	return applyToSelection(entryContent, "figure", func(s *goquery.Selection) {
		s.SetAttr("data-medium", "true")
	})
}

func useNoScriptImages(entryContent string) string {
	return applyToSelection(entryContent, "noscript", func(s *goquery.Selection) {
		if img := s.Find("img"); img.Length() > 0 {
			s.ReplaceWithHtml(img.Text())
		}
	})
}

func getYoutubVideoIDFromURL(entryURL string) string {
	parsed, err := url.Parse(entryURL)
	if err != nil {
		return ""
	}
	if id := parsed.Query().Get("v"); id != "" {
		return id
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func buildVideoPlayerIframe(absoluteVideoURL string) string {
	return `<iframe src="` + absoluteVideoURL + `"></iframe>`
}

func addVideoPlayerIframe(absoluteVideoURL, entryContent string) string {
	return buildVideoPlayerIframe(absoluteVideoURL) + `<br>` + entryContent
}

func addYoutubeVideoRewriteRule(entryURL, entryContent string) string {
	videoID := getYoutubVideoIDFromURL(entryURL)
	if videoID == "" {
		return entryContent
	}
	return addVideoPlayerIframe("https://www.youtube.com/embed/"+videoID, entryContent)
}

func addYoutubeVideoUsingInvidiousPlayer(entryURL, entryContent string) string {
	videoID := getYoutubVideoIDFromURL(entryURL)
	if videoID == "" {
		return entryContent
	}
	return addVideoPlayerIframe("https://yewtu.be/embed/"+videoID, entryContent)
}

func addYoutubeVideoFromId(entryContent string) string {
	return applyToSelection(entryContent, `iframe[src*="youtube.com"]`, func(s *goquery.Selection) {
		s.SetAttr("allowfullscreen", "true")
	})
}

func addInvidiousVideo(entryURL, entryContent string) string {
	videoID := getYoutubVideoIDFromURL(entryURL)
	if videoID == "" {
		return entryContent
	}
	return addVideoPlayerIframe("https://invidious.snopyta.org/embed/"+videoID, entryContent)
}

func addPDFLink(entryURL, entryContent string) string {
	if !strings.HasSuffix(strings.ToLower(entryURL), ".pdf") {
		return entryContent
	}
	return entryContent + `<br><a href="` + entryURL + `">PDF</a>`
}

func addEnclosureLinks(entry *model.Entry) string {
	var links []string
	for _, enclosure := range entry.Enclosures {
		links = append(links, `<a href="`+enclosure.URL+`">`+enclosure.URL+`</a>`)
	}
	return entry.Content + strings.Join(links, "<br>")
}

func replaceTextLinks(input string) string {
	return textLinkRegex.ReplaceAllString(input, `${1}<a href="${2}">${2}</a>`)
}

func replaceCustom(entryContent string, searchTerm string, replaceTerm string) string {
	re, err := regexp.Compile(searchTerm)
	if err != nil {
		return entryContent
	}
	return re.ReplaceAllString(entryContent, replaceTerm)
}

func removeCustom(entryContent string, selector string) string {
	return applyToSelection(entryContent, "body", func(s *goquery.Selection) {
		s.Find(selector).Remove()
	})
}

func addCastopodEpisode(entryURL, entryContent string) string {
	player := `<iframe width="650" frameborder="0" src="` + entryURL + `/embed/light"></iframe>`
	return player + `<br>` + entryContent
}

func applyFuncOnTextContent(entryContent string, selector string, repl func(string) string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(entryContent))
	if err != nil {
		return entryContent
	}

	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		if s.Nodes[0].Type == 3 {
			s.ReplaceWithHtml(repl(s.Nodes[0].Data))
		}
	})

	output, _ := doc.Find("body").Html()
	return output
}

func decodeBase64Content(entryContent string) string {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(entryContent))
	if err != nil {
		return entryContent
	}
	return html.EscapeString(string(decoded))
}

func addHackerNewsLinksUsing(entryContent, app string) string {
	return applyToSelection(entryContent, "a", func(s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href != "" {
			s.SetAttr("href", href+"?ref="+app)
		}
	})
}

func removeTables(entryContent string) string {
	return applyToSelection(entryContent, "table", func(s *goquery.Selection) {
		s.Remove()
	})
}

func fixGhostCards(entryContent string) string {
	return applyToSelection(entryContent, ".kg-card", func(s *goquery.Selection) {
		s.SetAttr("data-ghost-card", "true")
	})
}

func removeImgBlurParams(entryContent string) string {
	return applyToSelection(entryContent, "img", func(s *goquery.Selection) {
		src, _ := s.Attr("src")
		src = regexp.MustCompile(`[?&](blur|quality)=\d+`).ReplaceAllString(src, "")
		s.SetAttr("src", src)
	})
}
