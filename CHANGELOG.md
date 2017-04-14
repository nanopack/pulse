## v0.2.2 (Apr 14, 2017)

IMPROVEMENTS:
  - Allow integers or floats for threshold
  - Allow different comparison operators
  - Added routes for listing single and all tasks
  - Updated alerts API docs

## v0.2.1 (Feb 8, 2017)

BUG FIXES:
  - #10 Resolves client multiple heartbeats
  - #10 Resolves server's client-awareness issues
  - Fixed `beat-interval` default

IMPROVEMENTS:
  - Added dirty commit identifier

## v0.2.0 (Jan 30, 2017)

BUG FIXES:
  - Resolve connection timeout issue (for real this time)

IMPROVEMENTS:
  - Handle errors rather than panic

## v0.1.0 (Jan 3, 2017)

BUG FIXES:
  - Resolve connection timeout issue
  - Made alert ids unique

IMPROVEMENTS:
  - Added a better way to read from the connection so disconnects are detected
  - Created meaningful alert id and message
  - Alert on statechange only
  - Tests

## Previous (Jan 3, 2017)

This change log began with version 0.1.0. Any prior changes can be seen by viewing
the commit history.
