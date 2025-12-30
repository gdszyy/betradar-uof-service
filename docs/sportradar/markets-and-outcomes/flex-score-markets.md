# Flex Score Markets

There is a special type of market called flex score markets. These are only offered live and are an alternative to correct score markets.

The correct score markets have a fixed set of outcomes. If the score goes above the fixed set of outcomes the ordinary correct score market is deactivated. The flex score markets offer a set of outcomes based on the current score of the match, hence they are always available regardless of how many goals are scored.

These markets are displayed in the market description endpoint by having an extra `<attribute>`-element with the name `is_flex_score`. For these markets, the proper outcome names are created by taking the current score and “adding” the listed outcome name to that score.

**For example:**

> If the current score is 5:3 and the outcome name is 1:1. Betting on that outcome means betting on a final outcome of 6:4.
