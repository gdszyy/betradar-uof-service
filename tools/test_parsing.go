package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
)

func parseMessageOld(xmlContent string) string {
	var messageType string
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlContent)))
	token, _ := decoder.Token()
	if startElement, ok := token.(xml.StartElement); ok {
		messageType = startElement.Name.Local
	}
	return messageType
}

func parseMessageNew(xmlContent string) string {
	var messageType string
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlContent)))
	// 循环读取token直到找到第一个StartElement(跳过XML声明等)
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if startElement, ok := token.(xml.StartElement); ok {
			messageType = startElement.Name.Local
			break
		}
	}
	return messageType
}

func main() {
	testMessages := []struct {
		name string
		xml  string
	}{
		{
			name: "alive message",
			xml:  `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><alive product="3" timestamp="1761016043266" subscribed="1"/>`,
		},
		{
			name: "fixture_change message",
			xml:  `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><fixture_change start_time="1761948480000" next_live_time="1761775680000" product="1" event_id="sr:match:64158625" timestamp="1761016040499"/>`,
		},
		{
			name: "odds_change message (sample)",
			xml:  `<?xml version="1.0" encoding="UTF-8"?><odds_change product="1" event_id="sr:match:12345" timestamp="1234567890"><odds><market id="1" status="1"><outcome id="1" odds="1.5" active="1"/></market></odds></odds_change>`,
		},
		{
			name: "bet_stop message (sample)",
			xml:  `<?xml version="1.0" encoding="UTF-8"?><bet_stop product="1" event_id="sr:match:12345" timestamp="1234567890" market_status="1"/>`,
		},
		{
			name: "bet_settlement message (sample)",
			xml:  `<?xml version="1.0" encoding="UTF-8"?><bet_settlement product="1" event_id="sr:match:12345" timestamp="1234567890" certainty="1"><outcomes><market id="1"><outcome id="1" result="1"/></market></outcomes></bet_settlement>`,
		},
	}

	log.Println("=== Testing XML Parsing ===\n")

	for _, test := range testMessages {
		oldResult := parseMessageOld(test.xml)
		newResult := parseMessageNew(test.xml)

		fmt.Printf("Test: %s\n", test.name)
		fmt.Printf("  Old parsing: '%s' %s\n", oldResult, getStatus(oldResult))
		fmt.Printf("  New parsing: '%s' %s\n", newResult, getStatus(newResult))
		fmt.Println()
	}

	log.Println("=== Test Complete ===")
}

func getStatus(result string) string {
	if result == "" {
		return "❌ FAILED (empty)"
	}
	return "✓ SUCCESS"
}

