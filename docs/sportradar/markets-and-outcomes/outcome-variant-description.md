# Outcome variant description

Some markets and outcomes have variable descriptions. This is for example the case for many outright markets and outcomes, as well as some dynamic markets – most importantly the correct score market, but also some cricket markets. These markets have a special specifier `variant` and this specifier is set in the odds_change message to a urn that looks like this: `pre:markettext:1234`

To find the actual market name you need to do an additional API-call to `descriptions/en/markets/<market>/variant/<urn>`.

The returned document will return the market name and where applicable, outcome names. Note that the same market can have different market descriptions at different times if the variant-specifier changes.

## XML Example

```xml
<market_descriptions response_code="OK">
  <market id="241" name="Exact games" variant="sr:exact_games:bestof:5:39">
    <outcomes>
      <outcome id="sr:exact_games:bestof:5:39" name="3"/>
      <outcome id="sr:exact_games:bestof:5:40" name="4"/>
      <outcome id="sr:exact_games:bestof:5:41" name="5"/>
    </outcomes>
  </market>
</market_descriptions>
```

Two markets with the same market ID, but with different variant descriptions should be treated as different market lines. This is the same way it works otherwise (i.e. market-id + specifiers uniquely identify a market line). Consequently, the Betradar SDK will book and settle these market lines independently of each other.
