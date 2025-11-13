package services

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)



// ExtractEventIDFromURN 从 event URN (sr:match:123) 中提取数字 ID (123)
func ExtractEventIDFromURN(urn string) (int64, error) {
	parts := strings.Split(urn, ":")
	if len(parts) != 3 || parts[0] != "sr" || parts[1] != "match" {
		return 0, fmt.Errorf("invalid event URN format: %s", urn)
	}
	return strconv.ParseInt(parts[2], 10, 64)
}


// ExtractMarketIDFromURN 从 market URN (sr:market:123) 中提取数字 ID (123)
func ExtractMarketIDFromURN(urn string) (int64, error) {
	parts := strings.Split(urn, ":")
	if len(parts) != 3 || parts[0] != "sr" || parts[1] != "market" {
		return 0, fmt.Errorf("invalid market URN format: %s", urn)
	}
	return strconv.ParseInt(parts[2], 10, 64)
}

// CleanSQLQuery 清理 SQL 语句中的换行符和多余空格，使其成为单行
func CleanSQLQuery(query string) string {
	// 替换所有换行符和制表符为空格
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.ReplaceAll(query, "\t", " ")

	// 替换多个连续空格为一个空格
	for strings.Contains(query, "  ") {
		query = strings.ReplaceAll(query, "  ", " ")
	}

	// 去除首尾空格
	return strings.TrimSpace(query)
}

// XMLToJSONMap converts XML bytes to a map[string]interface{} suitable for JSON serialization.
// It uses a simplified approach to extract top-level attributes and the inner XML content.
func XMLToJSONMap(xmlBytes []byte) (map[string]interface{}, error) {
	// Define a simple struct to capture the root element's attributes and inner content
	type SimpleXML struct {
		XMLName xml.Name `xml:"-"`
		Attrs   []xml.Attr `xml:",any,attr"`
		Content string `xml:",innerxml"`
	}
	
	var s SimpleXML
	if err := xml.Unmarshal(xmlBytes, &s); err != nil {
		return nil, err
	}
	
	result := make(map[string]interface{})
	
	// Add attributes (e.g., @event_id, @product)
	for _, attr := range s.Attrs {
		result["@"+attr.Name.Local] = attr.Value
	}
	
	// Add inner XML as a string. This is the compromise for complexity,
	// allowing the frontend to still access the full XML content if needed,
	// but providing the key attributes in a structured way.
	result["xml_content"] = s.Content
	
	// Wrap the result with the root element name (e.g., odds_change)
	// We need to find the root tag name from the XML bytes
	rootName := ""
	if len(xmlBytes) > 0 {
		// Simple way to find the root tag name (e.g., <odds_change ...>)
		start := strings.Index(string(xmlBytes), "<")
		end := strings.Index(string(xmlBytes), " ")
		if start != -1 && end != -1 && end > start {
			rootName = string(xmlBytes[start+1 : end])
		}
	}
	
	if rootName != "" {
		return map[string]interface{}{rootName: result}, nil
	}
	
	return result, nil
}
