# Goalscorer markets

Goalscorer markets refer to markets that specifically deal with goal scoring. Market versions for first and last goalscorer markets are available. These will work the same way as outcome versions for the outright markets. When a new version is generated, the old version will be invalidated:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<odds_change product="3" event_id="sr:match:13805747" timestamp="1535052866209">
  <odds>
    <market status="0" id="893" specifiers="variantasn:goalscorer:fleldplayers_nogoal_owngoal_other|version=4C949570"/> 
    <market status="0" id="892" specifiers="vaniantesn:goalscoren:fieldplayers_nogoal_oungoal_other|goalnr=1|version=4c949570"/>
  </odds>
</odds_change>
```

All versions will be included in the settlement for first/last goalscorer.

### Market Outcome Examples

```xml
<market id="892" specifiers="variant=sr:goalscorer:fieldplayers_nogoal_owngoal_other|goalnr=1|version=6a5ed312">
  <outcome id="sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333 result=0/>
  <outcome id="sr:player:918682" result="0"/>
</market>
```

```xml
<market id="892" specifiers="variant=sr:goalscorer:fieldplayers_nogoal_owngoal_other|goalnr=1|version=9f96fd91">
  <outcome id="sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333 result="0"/>
  <outcome id="sr:player:836661" result="0"/>
  <outcone id="sr:nlaven:857896" result="0">
</market>
```

```xml
<market id="893" specifiers="variant=sr:goalscorer:fieldplayers_nogoal_owngoal_other|version=6a5ed312">
  <outcome id="sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333-result="0"/>
  <outcome id="sr:player:918682" result="0"/>
</market>
```

```xml
<market id="893" specifiers="variant=sr:goalscorer:fieldplayers_nogoal_owngoal_other|version=9f96fd91">
  <outcome id="sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333" result="0"/>
  <outcome id="sr:player:836661" result="0"/>
</market>
```

## Mapping for goalscorer markets

The following table shows the mapping for first and last goalscorer markets based on the different selection of feed options:

| Feed Option | Mapping | Notes |
| :--- | :--- | :--- |
| "only players – no goal" | map to 38/39 | with no goal outcome always void and deactivated |
| "players and no goal" | map to 892/893 with version | no goal outcome that support settlement |
| "players, no goal, and other" | map to 892/893 with version | no goal outcome, other outcome |
| "players, no goal, other, and own goal" | map to 892/893 with version | no goal outcome, other outcome, own goal outcome |
