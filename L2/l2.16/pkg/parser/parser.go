package parser

import (
	"io"
	"net/url"
	"regexp"
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
			switch n.Data {
			case "a", "area":
				// ссылки на страницы
				if href := getAttr(n, "href"); href != "" {
					absolute := normalizeLink(pageURL, href)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						pageLinks = append(pageLinks, absolute)
					}
				}

			case "link":
				// CSS, favicon, preload и другие ресурсы
				href := getAttr(n, "href")
				if href == "" {
					return
				}

				rel := getAttr(n, "rel")
				as := getAttr(n, "as")

				// проверяем все типы ресурсов
				if strings.Contains(rel, "stylesheet") ||
					strings.Contains(rel, "icon") ||
					strings.Contains(rel, "shortcut") ||
					strings.Contains(rel, "apple-touch-icon") ||
					strings.Contains(rel, "mask-icon") ||
					strings.Contains(rel, "preload") ||
					strings.Contains(rel, "prefetch") ||
					strings.Contains(rel, "manifest") ||
					as != "" {
					absolute := normalizeLink(pageURL, href)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}

			case "script":
				// JavaScript файлы
				if src := getAttr(n, "src"); src != "" {
					absolute := normalizeLink(pageURL, src)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}

			case "img", "image":
				// изображения (включая srcset)
				extractImageLinks(n, pageURL, &resourceLinks)

			case "iframe", "embed", "frame", "portal":
				// встроенный контент
				if src := getAttr(n, "src"); src != "" {
					absolute := normalizeLink(pageURL, src)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}

			case "source":
				// источники для picture, audio, video
				if src := getAttr(n, "src"); src != "" {
					absolute := normalizeLink(pageURL, src)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}
				if srcset := getAttr(n, "srcset"); srcset != "" {
					extractSrcsetLinks(srcset, pageURL, &resourceLinks)
				}

			case "audio", "video", "track":
				// медиа элементы
				if src := getAttr(n, "src"); src != "" {
					absolute := normalizeLink(pageURL, src)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}
				if src := getAttr(n, "poster"); src != "" {
					absolute := normalizeLink(pageURL, src)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}

			case "object":
				// встроенные объекты
				if data := getAttr(n, "data"); data != "" {
					absolute := normalizeLink(pageURL, data)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}

			case "picture":
				// picture содержит source и img внутри, они будут обработаны рекурсивно

			case "svg", "use":
				// SVG элементы
				if href := getAttr(n, "href"); href != "" {
					absolute := normalizeLink(pageURL, href)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}
				if xlink := getAttr(n, "xlink:href"); xlink != "" {
					absolute := normalizeLink(pageURL, xlink)
					if absolute != "" && !isExternalLink(pageURL, absolute) {
						resourceLinks = append(resourceLinks, absolute)
					}
				}

			case "meta":
				// мета-теги с путями (например, og:image)
				if property := getAttr(n, "property"); property != "" {
					if property == "og:image" || property == "og:image:url" ||
						property == "twitter:image" || property == "twitter:image:src" {
						if content := getAttr(n, "content"); content != "" {
							absolute := normalizeLink(pageURL, content)
							if absolute != "" && !isExternalLink(pageURL, absolute) {
								resourceLinks = append(resourceLinks, absolute)
							}
						}
					}
				}
			}

			// проверяем атрибут style на наличие url()
			if style := getAttr(n, "style"); style != "" {
				extractStyleUrls(style, pageURL, &resourceLinks)
			}
		}

		// рекурсивно обходим дочерние элементы
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return pageLinks, resourceLinks, nil
}

// ExtractCSSLinks извлекает ссылки из CSS контента
func ExtractCSSLinks(cssContent string, baseURL *url.URL) []string {

	var links []string

	// регулярные выражения для поиска url() в CSS
	urlRegex := regexp.MustCompile(`url\(\s*['"]?([^'"()]+)['"]?\s*\)`)
	importRegex := regexp.MustCompile(`@import\s+(?:url\()?['"]?([^'"()]+)['"]?(?:\))?`)

	// ищем все url()
	matches := urlRegex.FindAllStringSubmatch(cssContent, -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			link := match[1]
			// игнорируем data: URL, base64 и т.д.
			if !strings.HasPrefix(link, "data:") && !strings.HasPrefix(link, "#") {
				absolute := normalizeLink(baseURL, link)
				if absolute != "" && !isExternalLink(baseURL, absolute) {
					links = append(links, absolute)
				}
			}
		}
	}

	// ищем @import
	imports := importRegex.FindAllStringSubmatch(cssContent, -1)
	for _, match := range imports {
		if len(match) > 1 && match[1] != "" {
			link := match[1]
			absolute := normalizeLink(baseURL, link)
			if absolute != "" && !isExternalLink(baseURL, absolute) {
				links = append(links, absolute)
			}
		}
	}

	return links
}

// getAttr возвращает значение атрибута или пустую строку
func getAttr(n *html.Node, attrName string) string {

	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

// extractImageLinks извлекает ссылки из тегов img (включая srcset)
func extractImageLinks(n *html.Node, pageURL *url.URL, resourceLinks *[]string) {

	// обычный src
	if src := getAttr(n, "src"); src != "" {
		absolute := normalizeLink(pageURL, src)
		if absolute != "" && !isExternalLink(pageURL, absolute) {
			*resourceLinks = append(*resourceLinks, absolute)
		}
	}

	// srcset может содержать несколько изображений
	if srcset := getAttr(n, "srcset"); srcset != "" {
		extractSrcsetLinks(srcset, pageURL, resourceLinks)
	}
}

// extractSrcsetLinks извлекает ссылки из атрибута srcset
func extractSrcsetLinks(srcset string, pageURL *url.URL, resourceLinks *[]string) {

	// srcset может содержать несколько URL с дескрипторами: "image.jpg 1x, image-2x.jpg 2x"
	parts := strings.Split(srcset, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// берем первую часть до пробела (URL)
		if idx := strings.Index(part, " "); idx > 0 {
			urlStr := part[:idx]
			absolute := normalizeLink(pageURL, urlStr)
			if absolute != "" && !isExternalLink(pageURL, absolute) {
				*resourceLinks = append(*resourceLinks, absolute)
			}
		} else if part != "" {
			// если нет дескриптора, то весь part - это URL
			absolute := normalizeLink(pageURL, part)
			if absolute != "" && !isExternalLink(pageURL, absolute) {
				*resourceLinks = append(*resourceLinks, absolute)
			}
		}
	}
}

// extractStyleUrls извлекает url() из атрибута style
func extractStyleUrls(style string, pageURL *url.URL, resourceLinks *[]string) {

	// простой парсинг url() в стилях
	start := 0
	for {
		idx := strings.Index(style[start:], "url(")
		if idx == -1 {
			break
		}
		idx += start + 4 // позиция после "url("

		// ищем закрывающую скобку
		end := strings.Index(style[idx:], ")")
		if end == -1 {
			break
		}
		end += idx

		// извлекаем URL
		urlContent := style[idx:end]
		urlContent = strings.Trim(urlContent, " '\"")

		if urlContent != "" {
			absolute := normalizeLink(pageURL, urlContent)
			if absolute != "" && !isExternalLink(pageURL, absolute) {
				*resourceLinks = append(*resourceLinks, absolute)
			}
		}

		start = end + 1
	}
}

// normalizeLink преобразует ссылку в абсолютный URL
func normalizeLink(base *url.URL, link string) string {

	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}

	// игнорируем якоря, javascript, mailto, data: и т.д.
	if strings.HasPrefix(link, "#") ||
		strings.HasPrefix(link, "javascript:") ||
		strings.HasPrefix(link, "mailto:") ||
		strings.HasPrefix(link, "tel:") ||
		strings.HasPrefix(link, "data:") ||
		strings.HasPrefix(link, "blob:") ||
		strings.HasPrefix(link, "file:") {
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

// isExternalLink проверяет, является ли ссылка внешней
func isExternalLink(base *url.URL, link string) bool {

	parsed, err := url.Parse(link)
	if err != nil {
		return true
	}

	// если хост пустой или совпадает - это внутренняя ссылка
	if parsed.Host == "" || parsed.Host == base.Host {
		return false
	}

	return true
}
