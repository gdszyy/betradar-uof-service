# Market Mapping

This section covers the most relevant information in regards to markets, bookings and outcomes in Unified feed. If you are coming over from the legacy Live Odds feed, there are some notable changes to how markets work.

## General market mapping information

Mappings are an essential part of the Unified Odds Feed. In this section you will find some of the more common mappings used between different odds producers and the mapping between legacy products and the Unified Odds Feed.

*   **Product\_id**: Sometimes this can be mapped to multiple products/producers. An example would be that if the product\_id is 1|4, it is mapped to product 1 (Live Odds) and 4 (BetPal).
*   **Product\_outcome\_id**: Indicates to which outcome ID from the legacy feed this is mapped to.
*   **Sov\_template**: Shows the special odds value (sov) from the legacy feeds.

Here are some examples to illustrate how some of the mapping works:

### XML Example 1: Market 447 (Total for a period)

\`\`\`xml
<market id="447" name="{!periodnr} period - {$competitor1} total" groups="all">
    <outcomes>
        <outcome id="13" name="under {total}"/>
        <outcome id="12" name="over {total}"/>
    </outcomes>
    <specifiers>
        <specifier name="periodnr" type="integer"/>
        <specifier name="total" type="decimal"/>
    </specifiers>
    <mappings>
        <mapping product_id="1" product_ids="1|4" sport_id="sr:sport:4" market_id="8:1062" sov_template="{periodnr}/{total}">
            <mapping_outcome outcome_id="13" product_outcome_id="4860" product_outcome_name="under"/>
            <mapping_outcome outcome_id="12" product_outcome_id="4862" product_outcome_name="over"/>
        </mapping>
    </mappings>
</market>
\`\`\`

Market 447 (Unified feed ID) has a mapping for Live Odds, which is the same for Betpal. It maps to legacy Live Odds market 8:1062 (Total home team [total] for [periodNr!] period), which has special odds values `{periodnr}/{total}`. Outcome 13 (ID=”13”) of the new market maps to legacy Live Odds outcome id 4860 (under), and outcome 12 maps to 4863 (over).

### XML Example 2: Market 330 (Multiple Mappings)

Furthermore, some markets can map to multiple legacy markets, based on the value of their specifiers. In the following example you will be able to see how one market has 4 different mappings. One is for Live Odds and the remaining 3 are for pre-match, identified by their product IDs.

\`\`\`xml
<market id="330" name="{!mapnr} map - winner (incl. overtime)" groups="all">
    <outcomes>
        <outcome id="4" name="{$competitor1}"/>
        <outcome id="5" name="{$competitor2}"/>
    </outcomes>
    <specifiers>
        <specifier name="mapnr" type="integer"/>
    </specifiers>
    <mappings>
        <mapping product_id="1" product_ids="1|4" sport_id="sr:sport:109" market_id="7:1390" sov_template="{mapnr}">
            <mapping_outcome outcome_id="4" product_outcome_id="17" product_outcome_name="1"/>
            <mapping_outcome outcome_id="5" product_outcome_id="18" product_outcome_name="2"/>
        </mapping>
        <mapping product_id="3" product_ids="3" sport_id="sr:sport:109" market_id="204" valid_for="mapnr=1">
            <mapping_outcome outcome_id="4" product_outcome_id="1" product_outcome_name="1"/>
            <mapping_outcome outcome_id="5" product_outcome_id="3" product_outcome_name="2"/>
        </mapping>
        <mapping product_id="3" product_ids="3" sport_id="sr:sport:109" market_id="231" valid_for="mapnr=2">
            <mapping_outcome outcome_id="4" product_outcome_id="1" product_outcome_name="1"/>
            <mapping_outcome outcome_id="5" product_outcome_id="3" product_outcome_name="2"/>
        </mapping>
        <mapping product_id="3" product_ids="3" sport_id="sr:sport:109" market_id="343" valid_for="mapnr=3">
            <mapping_outcome outcome_id="4" product_outcome_id="1" product_outcome_name="1"/>
            <mapping_outcome outcome_id="5" product_outcome_id="3" product_outcome_name="2"/>
        </mapping>
    </mappings>
</market>
\`\`\`

### XML Example 3: Market 139 (Total bookings)

The below example shows how the market “Total bookings” is described. The mapping section specifies first that for Live Odds (Soccer), this market is known as type=”8” and subtype=”153” (product\_id=”1” for Live Odds). Then it says that for LCoO for Soccer (product\_id=”3”), that this market is known as 236.

\`\`\`xml
<market id="139" name="Total bookings">
  <outcomes>
    <outcome id="13" name="under {total}"/>
    <outcome id="12" name="over {total}"/>
  </outcomes>
  <specifiers>
    <specifier name="total" type="decimal"/>
  </specifiers>
  <mappings>
    <mapping product_id="1" sport_id="sr:sport:1" market_id="8:153" sov_template="{total}">
       <mapping_outcome outcome_id="12" product_outcome_id="573" product_outcome_name="over"/>
       <mapping_outcome outcome_id="13" product_outcome_id="571" product_outcome_name="under"/>
    </mapping>
    <mapping product_id="3" sport_id="sr:sport:1" market_id="236" sov_template="{total}">
       <mapping_outcome outcome_id="12" product_outcome_id="6" product_outcome_name="over"/>
       <mapping_outcome outcome_id="13" product_outcome_id="7" product_outcome_name="under"/>
    </mapping>
  </mappings>
</market>
\`\`\`

### XML Example 4: Market 241 (Exact Games)

The following excerpt for the market “Exact Games” unified odds id: 241. This says that this market is mapped to Live Odds type 8 and subtype 25 for the three sports: Table Tennis (sr:sport:20), Badminton (sr:sport:31) & Squash (sr:sport:37).

\`\`\`xml
<mapping product_id="1" sport_id="sr:sport:20" market_id="8:25" valid_for="variant=sr:exact_goals:bestof:5"/>
<mapping product_id="1" sport_id="sr:sport:31" market_id="8:25" valid_for="variant=sr:exact_goals:bestof:5"/>
<mapping product_id="1" sport_id="sr:sport:37" market_id="8:25" valid_for="variant=sr:exact_goals:bestof:5"/>
\`\`\`

### XML Example 5: Market 199 (Correct Score best of 3)

There are some noteworthy changes to keep in mind: In Unified Odds, two markets in two different sports that have the same meaning, have often been mapped to the same unified market. The following excerpt says that the market “Correct Score best of 3” (id 199) is mapped to id 233 for LCOO for Tennis, but to id 350 for LCOO for BeachVolley.

\`\`\`xml
<mapping product_id="3" sport_id="sr:sport:5" market_id="233"/>
<mapping product_id="3" sport_id="sr:sport:34" market_id="350"/>
\`\`\`

### XML Example 6: Market 303 (Xth quarter handicap)

Because of the new way unified odds uses specifiers, in some cases where we previously used multiple markets, Unified Odds now use one market with different specifiers. For example, in “Xth quarter handicap” – which used to be one market for each quarter in Live Odds, is now the same market with different specifiers in Unified Odds (Unified Odds market 303):

\`\`\`xml
<mapping product_id="1" sport_id="sr:sport:2" market_id="7:44" valid_for="quarternr=1"/>
<mapping product_id="1" sport_id="sr:sport:2" market_id="7:48" valid_for="quarternr=2"/>
<mapping product_id="1" sport_id="sr:sport:2" market_id="7:51" valid_for="quarternr=3"/>
<mapping product_id="1" sport_id="sr:sport:2" market_id="7:54" valid_for="quarternr=4"/>
\`\`\`

### XML Example 7: Totals markets with `valid_for` pattern

A special `valid_for` pattern is used for totals markets where certain multiple (\*.5) are mapped to the Live Odds normal total market, and other multipliers (\*.0, \*.25, \*.75) are mapped to the Live Odds Asian Total. The below example shows the pattern with the meaning for soccer, for Live Odds this market is called 7:21 if the total specifier ends with .5:

\`\`\`xml
<mapping product_id="1" sport_id="sr:sport:1" market_id="7:21" valid_for="total~*.5">
\`\`\`

## Market & outcome description - variant text

Some markets and outcomes have variable descriptions. This is for example the case for many outright markets and outcomes, as well as some dynamic markets – most importantly the correct score market, but also some cricket markets. These markets have a special specifier `variant` and this specifier is set in the `odds_change` message to a URN that looks like this: `pre:markettext:1234`.

To find the actual market name you need to do an additional API-call to `descriptions/en/markets/<market_id>/variant/<variant_urn>`. The returned document will return the market name and where applicable, outcome names. Note that the same market can have different market descriptions at different times if the variant-specifier changes.

### XML Example 8: Variant Text Description

\`\`\`xml
<market_descriptions response_code="OK">
    <market id="241" name="Exact games" variant="sr:exact_games:bestof:5">
        <outcomes>
            <outcome id="sr:exact_games:bestof:5:39" name="3"/>
            <outcome id="sr:exact_games:bestof:5:40" name="4"/>
            <outcome id="sr:exact_games:bestof:5:41" name="5"/>
        </outcomes>
    </market>
</market_descriptions>
\`\`\`

Two markets with the same market id but with different variant descriptions should be treated as different market lines. This is the same way it works otherwise (i.e. market-id + specifiers uniquely identify a market line). Consequently, the Betradar system will handle and settle these market lines independently of each other.
