package linkprocessor

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

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

// ReplaceCSSLinks заменяет все ссылки в CSS на локальные
func ReplaceCSSLinks(cssContent []byte, pageURL, baseURL *url.URL) ([]byte, error) {

	cssStr := string(cssContent)

	// заменяем url()
	re := regexp.MustCompile(`url\(\s*['"]?([^'"()]+)['"]?\s*\)`)
	result := re.ReplaceAllStringFunc(cssStr, func(match string) string {
		// извлекаем URL из url()
		submatch := re.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		link := submatch[1]
		// игнорируем data: URL и хэши
		if strings.HasPrefix(link, "data:") || strings.HasPrefix(link, "#") {
			return match
		}

		newLink := replaceLink(link, pageURL, baseURL)
		return strings.Replace(match, link, newLink, 1)
	})

	// заменяем @import
	importRe := regexp.MustCompile(`@import\s+(?:url\()?['"]?([^'"()]+)['"]?(?:\))?`)
	result = importRe.ReplaceAllStringFunc(result, func(match string) string {
		submatch := importRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		link := submatch[1]
		newLink := replaceLink(link, pageURL, baseURL)
		return strings.Replace(match, link, newLink, 1)
	})

	return []byte(result), nil
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
	localPath, err := getLocalPath(absoluteURL, baseURL)
	if err != nil {
		return link
	}

	// вычисляем локальный путь для текущей страницы
	currentPageLocalPath, err := getLocalPath(pageURL, baseURL)
	if err != nil {
		return link
	}

	// вычисляем относительный путь от текущей страницы до ресурса
	relativePath, err := getRelativePath(currentPageLocalPath, localPath)
	if err != nil {
		return link
	}

	return relativePath
}

// getLocalPath возвращает локальный путь для URL (без домена)
func getLocalPath(fileURL, baseURL *url.URL) (string, error) {

	if fileURL.Hostname() != baseURL.Hostname() {
		return "", fmt.Errorf("host mismatch")
	}

	path := fileURL.Path
	query := fileURL.RawQuery

	// если путь пустой или заканчивается на / — добавляем index.html
	if path == "" || strings.HasSuffix(path, "/") {
		path = path + "index.html"
	} else {
		// Для HTML-страниц без расширения ВСЕГДА добавляем .html
		// Проверяем по расширению - если нет точки, это HTML-страница
		if !strings.Contains(filepath.Base(path), ".") {
			path = path + ".html"
		}
	}

	// убираем начальный слэш
	path = strings.TrimPrefix(path, "/")

	// кодируем query параметры в имя файла, если они есть
	if query != "" {
		encodedQuery := strings.ReplaceAll(query, "?", "_")
		encodedQuery = strings.ReplaceAll(encodedQuery, "&", "_")
		encodedQuery = strings.ReplaceAll(encodedQuery, "=", "-")

		dir, file := filepath.Split(path)
		name := strings.TrimSuffix(file, filepath.Ext(file))
		ext := filepath.Ext(file)
		path = filepath.Join(dir, name+"_"+encodedQuery+ext)
	}

	return path, nil
}

// getRelativePath возвращает относительный путь от source к target
func getRelativePath(source, target string) (string, error) {

	// используем filepath для корректной работы с путями в текущей ОС
	// затем преобразуем разделители в / для веба

	// определяем директорию исходного файла
	sourceDir := filepath.Dir(source)
	if sourceDir == "." {
		sourceDir = ""
	}

	// вычисляем относительный путь
	rel, err := filepath.Rel(sourceDir, target)
	if err != nil {
		return "", err
	}

	// заменяем разделители пути на / для веба
	rel = filepath.ToSlash(rel)

	// если результат пустой или начинается с .., оставляем как есть
	// иначе добавляем ./ для файлов в той же директории
	if rel != "" && !strings.HasPrefix(rel, ".") && !strings.HasPrefix(rel, "/") {
		rel = "./" + rel
	}

	return rel, nil
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
