package parser

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ExtractLinks извлекает все ссылки из HTML документа
func ExtractLinks(pageURL *url.URL, body io.Reader) (pageLinks, resourceLinks []string, err error) {

	doc, err := html.Parse(body)
	if err != nil {
		return nil, nil, err
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			var linkAttr string
			var isPage, isResource bool

			switch n.Data {
			case "a":
				linkAttr = "href"
				isPage = true
			case "link":
				// проверяем rel="stylesheet"
				for _, attr := range n.Attr {
					if attr.Key == "rel" && attr.Val == "stylesheet" {
						linkAttr = "href"
						isResource = true
						break
					}
				}
			case "script":
				linkAttr = "src"
				isResource = true
			case "img":
				linkAttr = "src"
				isResource = true
			case "iframe", "embed", "source", "track":
				linkAttr = "src"
				isResource = true
			case "video", "audio":
				// у video/audio может быть src, но также source внутри
				linkAttr = "src"
				isResource = true
			}

			// если нашли подходящий атрибут — извлекаем ссылку
			if linkAttr != "" {
				for _, attr := range n.Attr {
					if attr.Key == linkAttr {
						link := attr.Val
						absolute := normalizeLink(pageURL, link)
						if absolute == "" {
							continue
						}
						if isPage {
							pageLinks = append(pageLinks, absolute)
						} else if isResource {
							resourceLinks = append(resourceLinks, absolute)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return pageLinks, resourceLinks, nil
}

// normalizeLink преобразует ссылку в абсолютный URL
func normalizeLink(base *url.URL, link string) string {

	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}

	// игнорируем якоря, javascript, mailto и т.д.
	if strings.HasPrefix(link, "#") ||
		strings.HasPrefix(link, "javascript:") ||
		strings.HasPrefix(link, "mailto:") ||
		strings.HasPrefix(link, "tel:") {
		return ""
	}

	absURL, err := base.Parse(link)
	if err != nil {
		return ""
	}

	// убираем фрагмент
	absURL.Fragment = ""

	return absURL.String()
}
