Config = {}

-- Chat command that opens the calendar.
Config.Command = 'halloween'

-- Month the calendar runs in (10 = October). Day N of the calendar unlocks at
-- midnight server time on day N of this month.
Config.Month = 10

-- Year the claims are tracked under. nil = current year, so the calendar
-- resets automatically next October.
Config.Year = nil

-- How many extra days a treat stays claimable after its own day.
-- 1 = today's treat plus yesterday's as a grace period. 0 = today only.
-- Anything older is missed for good.
Config.GraceDays = 1

-- 'citizenid' = one calendar per character.
-- 'license'   = one calendar per Rockstar license, shared by all characters.
Config.TrackBy = 'citizenid'

-- Testing only: set to a number (1-31) to pretend it is that day of the month.
-- Leave nil on the live server.
Config.DebugDay = nil

-- Chat message shown to a player when their character loads.
Config.JoinMessage = true

-- Chat message broadcast to everyone on an interval.
Config.Broadcast = {
    enabled = true,
    intervalMinutes = 60,
}

Config.ChatPrefix = 'Halloween'
Config.ChatColor = { 255, 122, 24 }

-- One entry per day. Every item name here exists in ox_inventory.
--   { item = 'name', count = n }  -> gives an inventory item
--   { money = n }                 -> gives cash through qbx_core
Config.Days = {
    [1]  = { { item = 'water', count = 3 }, { item = 'sandwich', count = 2 } },
    [2]  = { { money = 500 }, { item = 'coffee', count = 2 } },
    [3]  = { { item = 'burger', count = 2 }, { item = 'sprunk_drink', count = 2 } },
    [4]  = { { item = 'bandage', count = 5 } },
    [5]  = { { item = 'firework1', count = 2 } },
    [6]  = { { item = 'cooked_pizza', count = 1 }, { item = 'cola_drink', count = 2 } },
    [7]  = { { money = 1000 } },
    [8]  = { { item = 'gauze', count = 5 }, { item = 'firstaid', count = 1 } },
    [9]  = { { item = 'fishing_rod', count = 1 } },
    [10] = { { item = 'grape', count = 5 }, { item = 'grapejuice', count = 3 } },
    [11] = { { item = 'firework2', count = 2 } },
    [12] = { { item = 'cooked_fries', count = 2 }, { item = 'orange_drink', count = 2 } },
    [13] = { { item = 'WEAPON_FLASHLIGHT', count = 1 } },
    [14] = { { money = 1500 } },
    [15] = { { item = 'repair_kit', count = 2 } },
    [16] = { { item = 'wine', count = 2 } },
    [17] = { { item = 'firework3', count = 2 } },
    [18] = { { item = 'fishing_kit', count = 1 } },
    [19] = { { item = 'firstaid', count = 2 }, { item = 'bandage', count = 5 } },
    [20] = { { money = 2000 } },
    [21] = { { item = 'beer', count = 6 } },
    [22] = { { item = 'metal_detector', count = 1 } },
    [23] = { { item = 'firework4', count = 2 } },
    [24] = { { item = 'burger', count = 3 }, { item = 'cola_drink', count = 3 } },
    [25] = { { money = 2500 } },
    [26] = { { item = 'WEAPON_KNIFE', count = 1 } },
    [27] = { { item = 'vodka', count = 1 }, { item = 'whiskey', count = 1 } },
    [28] = { { item = 'repair_kit', count = 3 }, { item = 'gauze', count = 5 } },
    [29] = {
        { item = 'firework1', count = 1 }, { item = 'firework2', count = 1 },
        { item = 'firework3', count = 1 }, { item = 'firework4', count = 1 },
    },
    [30] = { { money = 5000 } },
    -- Halloween grand prize
    [31] = {
        { money = 10000 },
        { item = 'firework1', count = 3 }, { item = 'firework2', count = 3 },
        { item = 'firework3', count = 3 }, { item = 'firework4', count = 3 },
        { item = 'cooked_pizza', count = 2 }, { item = 'sprunk_drink', count = 4 },
    },
}
