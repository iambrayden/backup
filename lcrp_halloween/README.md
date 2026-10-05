# lcrp_halloween

Halloween countdown and daily reward calendar for Last Chance Roleplay (Qbox + ox_inventory).

## Features

- `/halloween` opens a NUI calendar with a live countdown to Halloween and to the next treat.
- 31 daily rewards (Oct 1–31). Day 31 is the grand prize.
- Missed days can be claimed later in the month (`Config.AllowCatchUp`).
- Chat message when a character loads, telling the player how many treats are waiting.
- Chat broadcast to everyone every hour (`Config.Broadcast`).
- Claims are stored in MySQL per Rockstar license by default, so alt characters can't claim twice.
- Every claim is checked on the server. Rewards are only given when the claim row is inserted, so double clicks or spammed requests can't pay out twice.

## Install

1. Drop `lcrp_halloween` into your resources folder.
2. Add `ensure lcrp_halloween` to `server.cfg` **after** `ox_lib`, `oxmysql`, `qbx_core` and `ox_inventory`.
3. Restart. The `lcrp_halloween_claims` table is created automatically (`sql/halloween.sql` is there if you'd rather run it by hand).

## Config

Everything lives in `config.lua`:

| Option | Default | What it does |
|---|---|---|
| `Command` | `halloween` | Chat command that opens the calendar |
| `AllowCatchUp` | `true` | Allow claiming missed days later in October |
| `TrackBy` | `license` | `license` = one calendar per player, `citizenid` = per character |
| `DebugDay` | `nil` | Set to 1–31 to test a specific day. **Leave nil on live.** |
| `JoinMessage` | `true` | Chat message when a character loads |
| `Broadcast` | every 60 min | Server-wide chat reminder |
| `Days` | 31 entries | Rewards: `{ item = 'name', count = n }` or `{ money = n }` |

Days unlock at midnight **server time**. On start, the server prints a warning for any reward item that ox_inventory doesn't recognize.

Item icons come from `ox_inventory/web/images/<item>.png`. An emoji is shown if an image is missing.
