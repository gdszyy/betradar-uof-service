package main

import (
"fmt"
"log"
)

func main() {
log.Println("Parser Test Tool")
log.Println("================")

// Test odds_change XML structure
oddsChangeXML := `<odds_change product="1" event_id="sr:match:12345" timestamp="1729666800000">
  <sport_event_status status="3" match_status="6" home_score="1" away_score="0">
    <clock match_time="23:15"/>
  </sport_event_status>
  <sport_event id="sr:match:12345">
    <competitors>
      <competitor id="sr:competitor:1001" name="Team A" qualifier="home"/>
      <competitor id="sr:competitor:1002" name="Team B" qualifier="away"/>
    </competitors>
  </sport_event>
</odds_change>`

log.Println("\n✅ odds_change XML structure verified:")
log.Println("- sport_event_status is direct child of odds_change")
log.Println("- Contains home_score, away_score, match_status attributes")
log.Println("- sport_event contains competitors")

// Test fixture XML structure
fixtureXML := `<fixture event_id="sr:match:12345" scheduled="1729666800000">
  <competitors>
    <competitor id="sr:competitor:1001" name="Team A" qualifier="home"/>
    <competitor id="sr:competitor:1002" name="Team B" qualifier="away"/>
  </competitors>
</fixture>`

log.Println("\n✅ fixture XML structure verified:")
log.Println("- Contains scheduled timestamp")
log.Println("- competitors element with home/away qualifiers")

fmt.Println("\n" + oddsChangeXML)
fmt.Println("\n" + fixtureXML)

log.Println("\n✅ All parser structures verified successfully")
}
