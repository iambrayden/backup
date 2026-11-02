# qbx_ratio

Shows the **LEO / Fire-EMS / Civilian** split of players on a Qbox (qbx_core) FiveM server.

- Pops up in the **bottom-right** for the first **60 seconds** after a player picks a character.
- Players can type **`/ratio`** at any time to show it again (15 seconds by default).
- Auto-refreshes every 5 seconds while visible.

## Install

1. Copy the `qbx_ratio` folder into your `resources` directory.
2. Add to `server.cfg` **after** `ox_lib` and `qbx_core`:
   ```
   ensure qbx_ratio
   ```

Dependencies: `qbx_core`, `ox_lib` (both are standard in every Qbox install).

## How players are classified

All counting is done **server-side** from qbx_core's player data. Clients only receive the totals,
never other players' jobs.

1. The server loops `exports.qbx_core:GetQBPlayers()`. That only includes players who have
   **loaded a character**, so people in the loading screen or on character select are not counted.
2. For each player it reads `PlayerData.job`:
   - If `job.name` is in a category's `jobNames` → that category.
   - Otherwise, if `job.type` is in a category's `jobTypes` → that category.
     Qbox's default `shared/jobs.lua` gives `police` type `leo` and `ambulance` type `ems`,
     so those work with no extra setup.
   - Anything else → **Civilian**.
3. If `Config.RequireOnDuty = true` (default), an **off-duty** cop/medic counts as a civilian,
   because they aren't working at the moment.

Got custom departments (BCSO, SASP, SAFD…)? Add their job names to `config.lua`,
or give them `type = 'leo'` / `type = 'ems'` / `type = 'fire'` in `qbx_core/shared/jobs.lua`.

## Config

See `config.lua`: display durations, command name, refresh rate, on-duty requirement,
category labels/colors/jobs.
