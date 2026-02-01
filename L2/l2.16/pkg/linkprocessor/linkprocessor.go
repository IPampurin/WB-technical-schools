package linkprocessor

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"mywget/pkg/filesystem"

	"golang.org/x/net/html"
)

// ReplaceLinks заменяет все ссылки в HTML на локальные
func ReplaceLinks(htmlContent []byte, pageURL, baseURL *url.URL) ([]byte, error) {

	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// список атрибутов, которые могут содержать ссылки
			attrs := []string{"href", "src", "srcset", "data-src", "action", "background", "poster"}
			for _, attr := range attrs {
				replaceAttr(n, attr, pageURL, baseURL)
			}
			// обработка стилей (style attribute)
			if n.Data == "style" {
				replaceStyleContent(n, pageURL, baseURL)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, fmt.Errorf("render HTML: %w", err)
	}

	return buf.Bytes(), nil
}

// replaceAttr заменяет ссылку в атрибуте
func replaceAttr(n *html.Node, attrName string, pageURL, baseURL *url.URL) {

	for i, attr := range n.Attr {
		if attr.Key == attrName {
			// в srcset может быть несколько ссылок, разделенных запятыми
			if attrName == "srcset" {
				newVal := replaceSrcset(attr.Val, pageURL, baseURL)
				n.Attr[i].Val = newVal
			} else {
				newVal := replaceLink(attr.Val, pageURL, baseURL)
				n.Attr[i].Val = newVal
			}
		}
	}
}

// replaceLink заменяет одну ссылку
func replaceLink(link string, pageURL, baseURL *url.URL) string {

	if link == "" {
		return link
	}

	// игнорируем якоря, javascript, mailto и т.д.
	if strings.HasPrefix(link, "#") ||
		strings.HasPrefix(link, "javascript:") ||
		strings.HasPrefix(link, "mailto:") ||
		strings.HasPrefix(link, "tel:") {
		return link
	}

	// парсим ссылку относительно текущей страницы
	absoluteURL, err := pageURL.Parse(link)
	if err != nil {
		return link
	}

	// если ссылка ведет на другой домен, оставляем как есть
	if absoluteURL.Hostname() != baseURL.Hostname() {
		return link
	}

	// вычисляем локальный путь для абсолютного URL
	// (для замены ссылок не важно, HTML это или нет, поэтому передаем пустой contentType)
	localPath, err := filesystem.GetLocalPath(absoluteURL, baseURL, "")
	if err != nil {
		return link
	}

	// вычисляем относительный путь от текущей страницы до локального пути
	currentPageLocalPath, err := filesystem.GetLocalPath(pageURL, baseURL, "")
	if err != nil {
		return link
	}

	relativePath, err := getRelativePath(currentPageLocalPath, localPath)
	if err != nil {
		return link
	}

	return relativePath
}

// replaceSrcset заменяет ссылки в атрибуте srcset
func replaceSrcset(srcset string, pageURL, baseURL *url.URL) string {

	parts := strings.Split(srcset, ",")
	for i, part := range parts {
		subparts := strings.Split(strings.TrimSpace(part), " ")
		if len(subparts) > 0 {
			link := subparts[0]
			newLink := replaceLink(link, pageURL, baseURL)
			subparts[0] = newLink
			parts[i] = strings.Join(subparts, " ")
		}
	}

	return strings.Join(parts, ", ")
}

// replaceStyleContent заменяет url() в стилях
func replaceStyleContent(n *html.Node, pageURL, baseURL *url.URL) {

	if n.Type == html.ElementNode && n.Data == "style" && n.FirstChild != nil {
		content := n.FirstChild.Data
		newContent := replaceCSSUrls(content, pageURL, baseURL)
		n.FirstChild.Data = newContent
	}
}

// replaceCSSUrls заменяет url() в CSS
func replaceCSSUrls(css string, pageURL, baseURL *url.URL) string {

	// просто заменяем (можно улучшить с помощью парсера CSS ?)
	// ищем все вхождения url(...)
	start := 0
	var result strings.Builder
	for {
		idx := strings.Index(css[start:], "url(")
		if idx == -1 {
			result.WriteString(css[start:])
			break
		}
		idx += start
		result.WriteString(css[start:idx])
		start = idx + 4 // длина "url("
		// ищем закрывающую скобку
		end := strings.Index(css[start:], ")")
		if end == -1 {
			result.WriteString(css[idx:])
			break
		}
		end += start
		urlContent := css[start:end]

		// убираем кавычки и пробелы
		urlContent = strings.Trim(urlContent, " '\"")
		newURL := replaceLink(urlContent, pageURL, baseURL)
		result.WriteString("url(")
		result.WriteString(newURL)
		result.WriteString(")")
		start = end + 1
	}

	return result.String()
}

// getRelativePath возвращает относительный путь от source к target
func getRelativePath(source, target string) (string, error) {

	// source и target должны быть абсолютными путями относительно корня сайта
	// например: source = "example.com/path/to/index.html", target = "example.com/static/style.css"
	// нужно убрать общий префикс (домен) и вычислить относительный путь

	// разделяем на компоненты
	sourceDir := filepath.Dir(source)
	targetFile := target

	// вычисляем относительный путь
	rel, err := filepath.Rel(sourceDir, targetFile)
	if err != nil {
		return "", err
	}

	// заменяем разделители пути на / для веба
	rel = filepath.ToSlash(rel)

	return rel, nil
}
