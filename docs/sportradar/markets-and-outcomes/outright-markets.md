# Outright markets

Outrights are ordinary markets in Unified Odds. Hence odds updates are provided with ordinary `odds_change` messages, and resulting is provided with normal `bet_settlement` messages.

Outright markets are markets that have the “outcome_type” attribute set to “pre:outcometext”. This helps to identify the outright markets. See [outcome variant description](link-to-outcome-variant-description) section for more information about market descriptions.

An outright is seen as one market for an event. The event can be a match, a race or a tournament season. This means the `event_id` is in one of the following formats: **sr:season:1234**, **sr:stage:1234** or **sr:simple_tournament**. The types are a bit different, but they can all be looked up using `sport_events/(id)/fixture.xml`. This will return either a `tournament_info` (in the case of an `sr:season`, some `sr:stages`, and `sr:simpletournament`) or a `fixtures_fixture`.

1.  **sr:season:1234** refers to a particular season of a tournament/league. This is the event id type you will most often see used for long-term outrights such as “Who is the winner of Premier League 1993 or 1994”.

    **XML Example**

    ```xml
    <tournament_seasons>
     <seasons>
        <season id="sr:season:55" name="Premier League 93/94" start_date="1993-08-14" end_date="1994-05-09" year="93/94" tournament_id="sr:tournament:17"/>
     
        <season id="sr:season:54" name="Premier League 94/95" start_date="1994-08-20" end_date="1995-05-14" year="94/95" tournament_id="sr:tournament:17"/>
     
        ... Information about all seasons ...
     </seasons>
    </tournament_seasons>
    ```

2.  **sr:stage:1234** is used for top-level race events (e.g. F1-season), but also for the individual competitions (F1 GP Monaco), as well as the actual race-events (F1 Qualification Race).

    **XML Example**

    ```xml
    <tournament id="sr:stage:306297" scheduled="2016-10-13T09:00:00+02:00" scheduled_end="2017-10-02T06:00:00+02:00" name="PGA Tour 2017">
    ```

3.  **sr:simple_tournament** does not have season information, and typically has less information like lack of translations. `sr:simple_tournament` is currently used for e.g. Golf tournaments.

    **XML Example**

    ```xml
    <tournament id="sr:simple_tournament:40" name="Eerste Divisie">
        <tournament_length start_date="2003-07-01" end_date="2042-12-15"/>
        <sport id="sr:sport:1" name="Soccer"/>
        <category id="sr:category:35" name="Netherlands" country_code="NLD"/>
    </tournament>
    ```

Some outrights the Betradar system provides in structured format: This means that the market has a fixed name, and that the outcomes are well-defined entities. Other outrights are free-text outrights. These outrights have a free-text name and free-text outcomes, and it is not so easy to determine if the outcome in one outright is the same as an outcome in another outright. It is Betradar’s intent to provide more and more outrights in the structured format in the future.

Outrights with free-text market name and outcome descriptions are handled by a system called [_variant descriptions_]().

**Outright XML example:**

```xml
<odds_change product="3" event_id="sr:season:41266" timestamp="1512039661932" request_id="642330526">
  <odds>
    <market favourite="1" status="1" id="534" specifiers="variant=pre:markettext:45810">
      <market_metadata next_betstop="1512157500000"/>
      <outcome id="pre:outcometext:5982" odds="30.0" active="1"/>
      <outcome id="pre:outcometext:6076" odds="1.1" active="1"/>
      <outcome id="pre:outcometext:6090" active="0"/>
      <outcome id="pre:outcometext:115271" odds="8.0" active="1"/>
      <outcome id="pre:outcometext:7306474" odds="7.0" active="1"/>
      <outcome id="pre:outcometext:7333812" active="0"/>
    </market>
  </odds>
</odds_change>
```

All outrights have similar format, the only exception is the **multiwinner** market. This market would have a specifier like `“variant=pre:markettext:45810|winners=3”`, where the additional specifier denotes the number of winners (`winners=3`).

## Outright market IDs

| ID | Name | Sport | Example markets |
| :--- | :--- | :--- | :--- |
| **559** | Free text market | Formula1, cycling, all race winter sports (but not ice hockey, etc.) | <ul><li>10 km Sprint oberhof – Winner</li><li>Tour de Suisse 2017 – Stage 2 – Winner</li><li>Azerbaijan Grand Prix Race Winning Constructor</li><li>Azerbaijan Grand Prix Race – Winning Margin</li><li>Azerbaijan Grand Prix Race – Fastest Lap Winner</li><li>Tour De France 2017 – Stage 3 – Winner</li><li>Tour De France 2017 - Winner</li></ul> |
| **535** | Short term free text market | All sports not covered by ID: 559 | <ul><li>Oscar – Best Actress</li><li>BMW SA Open 2017 – 1st Round Leader</li><li>Nobel Peace Prize 2017 - Winner</li></ul> |
| **534** | Championship free text market | All sports not covered by ID: 559 | <ul><li>WC 2018 Qualification UEFA Group A – Winner</li><li>ATP - Australian Open 2017 – Winner</li><li>Championship 2016/17 - Top Midlands Club</li><li>Ligue 1 2016/17 - w/o Paris Saint Germain – Winner</li><li>Bundesliga 2016/17 - To Finish Bottom</li></ul> |
| **536** | Free text multiwinner market | All sports | <ul><li>Premier League 2016/17 – Relegation</li><li>Ligue 1 2016/17 - Bottom 3</li><li>Bundesliga 2017/18 - w/o FC Bayern Munich - Top 3</li><li>7.5 km Sprint Women Oberhof - Top 3</li><li>Rally Sweden 2017 - Top 3</li><li>IAAF World Championships 2017 - Javelin Throw Men - Top 3</li></ul> |

## Outrights using the API

If you are using the APIs directly, the endpoint `sport_events/(id)/summary.xml` ([https://iodocs.betradar.com/](https://iodocs.betradar.com/)) will work for all types of events (including seasons, stages, tournaments, etc.) and will return different types of information for different events and for outrights. You can use this to find the name of the event as well as additional information where applicable.

## Outcome version for Outright markets & Goalscorer markets

For some markets the outcomes are not fixed, but they can vary over time e.g. First Goal Scorer market. There might be a new player added that wasn't there when the market was added, or an outrights winner market may receive a new competitor as an outcome that was not present when the outright market first was offered.

By default, in Unified Odds Feed these new outcomes are just added. However, there is a configurable option to turn on "**version**". When this option is turned on the outcomes for a particular market is fixed, and in examples such as the ones outlined below, a new market will be offered instead. To separate these different markets when this option is turned on there is an additional specifier added to this type of markets called **version**. Each market version will have its own **version id**.

*   When a new outcome is detected, the odds producer will deactivate the previous version of that market and send out the new market version.
*   During bet settlement, the odds producer will send out bet settlement for all the different markets that it has offered odds on.
*   Finally, during recovery, previously sent market versions will also be included, so that you can deactivate these market versions if they are still open on the client side.

On each change of the list of outcomes (outcome added & outcome removed), the system stops the current version of the market (outright markets), and generates a new version of the market (note that market status is changed to 0).

One important detail with the free text outrights, is that there are some free text outrights where other outcomes are used, and thereby versions don't apply. For example; where outcomes are "yes" and "no". These outright are not affected by turning on this option. So we would only offer "other" in a situation where there is a theoretical possibility for another winner (theoretical being a f1 grand prix race where maybe a driver can get sick etc.)

Settlements will still be done for all versions for one specific market, and only one version will be active at the same time.

You can also configure to add the “**other**” outcome to all outrights where the field of participants is not complete. When participants change on such a market, we provide a version number.

It is important to note here that that different market IDs will be used when turning on the “other” outcome. An example would be a last goalscorer market having the ID of 39, will be sent as ID 893 when the option is selected.

Settlements will still be done for all versions for one specific market, and only one version will be active at the same time:

```xml
<market id="893" specifiers="variant=sr:goalscorer:fieldplayers_nogoal_owngoal_other|version=9F96fd91">
  <outcome id="sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333" result="0"/>
  <outcome id="sr:player:836661" result="0"/>
</market>
```

This helps you to distinguish the outright markets where participants have changed, and helps you identifying the most recent one. It guarantees that all versions of such an outright are settled accordingly when you use the “other” outcome.

All versions will also be included in a recovery request.

### Market mapping for outright markets with outcome versions

The unified feed ID open market = market with version specifier:

| Market name | UF ID (original ID) | UF ID open market |
| :--- | :--- | :--- |
| Championship outright | 534 | 906 |
| Short term outright | 535 | 907 |
| Multiwinner outright | 536 | 908 |
| First goalscorer | 38 | 892 |
| Last goalscorer | 39 | 893 |

Please NOTE when changing this configuration!

Adding or removing outcomes to the Outright or Goalscorer markets by changing the respective configuration in the "Feed Options" results in creating new market IDs (see the mapping section) with the new set of outcomes.

Example Outrights:
*   ID 534 phased out market ID without "Other"
*   ID 906 new market ID including "Other"

It does NOT close the old market ID 534, and does NOT send any more updates to this market. These phased out markets have to be closed and settled manually. The same behavior applies when you change the configuration e.g. back from including "Other" to excluding "Other". The phased out market ID does not close and remains without any future updates.

This logic also applies when adding or removing the outcomes "No Goal", "Own Goal" or "Other" to Goalscorer markets.

Example First Goalscorer:
*   ID 38 phased out market ID
*   ID 892 new market ID

## Dead heat factor in outrights

Dead heat factor is a floating point number used when there is a tie in outrights. Payout should be calculated as:

`payout = stake * odds * dead_heat`

So in the event of a lot of a big tie in the outright, a punter can actually lose money on a winning bet.

**Examples:**

*   2-way tie for first place in an outright with one winner: dead heat would be **0.5**.
*   3-way tie for second place in an outright with three winners: Dead heat for 1st place is null, for the 3 tied players it is **0.33**.

The number of positions in this market is 1 as we only talk about 1st place in the group. The number of players tied for that position could be 2 or 3. Results are entered as the total score for the player for the tournament. The lower the score the better.

*   **Example 1:** So using the rule Number of positions tied for divided by number of players tied in that position = 1/2 = dead heat void factor = **0.5**.
*   **Example 2:** So using the rule number of positions tied for divided by number of players tied in that position = 1/3 = dead heat void factor = **0.33**.

## Outright Meta Data

Each outright contains metadata about the outright.

These attributes are:

*   `next_betstop`: the next time this outright market will be closed for betting. Many outright markets are closed for betting during the next round of matches etc. and then odds are updated after the matches and market is re-opened for betting. This is specified in UNIX time.
*   `start_time`: the date/time the event will start. This is specified in UNIX time.
*   `end_time`: the date/time the event will end. This is specified in UNIX time.
*   `aams_id`: the AAMS market id for this outright (this is required for customers in the Italian region). If there is no AAMS id available for an outright market then it will display "0".

An example of how this appears is here:

**XML example**

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<odds_change product="3" event_id="sr:season:80396" timestamp="1611736115411">
  <odds>
    <market status="1" id="906" specifiers="variant=pre:markettext:106414|version=bcc52f2d">
      <market_metadata next_betstop="1612155600000" start_time="1609763400000" end_time="1617285600000" aams_id="1544991"/>
      <outcome id="pre:outcometext:25980" odds="28.0" probabilities="0.02247177294310169" active="1"/>
    </market>
  </odds>
</odds_change>
```
