# Markets and Outcomes

This section covers the most relevant information related to markets and outcomes in the Unified feed. If you are coming over from the legacy Live Odds feed, there are some notable changes to how markets work. 

The markets are the betting proposition with outcomes on which punters place their bets. Different types of markets are defined for different matches. 

## The minutes before an event starts

A few minutes before an event starts (in which Betradar offers Live Odds), Betradar has scouts and operators in place monitoring the game. At this time, the live markets are opened. Up until the match scheduled start (or actual start), Betradar provides many markets for both live odds and pre-match odds. 

Betradar, by default, distributes the "live" odds as the odds for the market. You also have the option to use the pre-match odds instead. When the match starts (status="live") a pre-match only system must act on this message and close the markets.

If Betradar does not offer live coverage for a match, Betradar systems will send out a bet stop that closes the markets for this sport event once the scheduled start time of the event is exceeded. In addition, for matches without live coverage, Betradar will not update the status field to live during the match, the status field will change from `not_started` to `closed` when the results are entered.

## Special market statuses during handover

During the handover from the prematch producer to the Live Odds producer, the live producer will inform the prematch producer what markets it will provide, and then start sending odds for these markets to the client system. The prematch producer will send a final `odds_change` message. There are no odds updates in this message, only market status updates, the statuses will either be `deactivated` or `handed_over`.

The deactivated markets should be deactivated by the client system, as these are markets that the live producer will no longer send. The markets that are marked as `handed_over` should be handled the following way:

1. If you have already received odds for this market from the live odds producer, it should ignore this market update completely.
2. If it has not yet received odds for this market it should suspend that market.

This last step is to ensure consistency on the client side in the unlikely event that the live odds producer fails after it has told the prematch producer to stop sending odds, but before it has been able to send its first odds update for this sport event. In such a case: The markets will be suspended as they should, and as soon as the Live Odds producer is available again it will start sending odds for this market.
