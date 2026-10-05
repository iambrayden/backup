local TOTAL_DAYS = #Config.Days
local RESOURCE = GetCurrentResourceName()

local busy = {}
local welcomed = {}
local itemCache = {}

---------------------------------------------------------------------------
-- Helpers
---------------------------------------------------------------------------

local function itemData(name)
    if itemCache[name] == nil then
        itemCache[name] = exports.ox_inventory:Items(name) or false
    end
    return itemCache[name] or nil
end

local function chat(target, message)
    TriggerClientEvent('chat:addMessage', target, {
        color = Config.ChatColor,
        multiline = true,
        args = { ('🎃 %s'):format(Config.ChatPrefix), message },
    })
end

local function getIdentifier(src)
    if Config.TrackBy == 'citizenid' then
        local player = exports.qbx_core:GetPlayer(src)
        return player and player.PlayerData.citizenid
    end
    return GetPlayerIdentifierByType(src, 'license2') or GetPlayerIdentifierByType(src, 'license')
end

--- Returns year, calendar day index, and phase ('before' | 'active' | 'after').
--- The index counts days from day 1 of Config.Month, so it is <= 0 before the
--- calendar opens and > TOTAL_DAYS after it ends (e.g. 32 on November 1st).
local function calendarNow()
    local t = os.date('*t')
    local year = Config.Year or t.year

    local index
    if Config.DebugDay then
        index = Config.DebugDay
    else
        -- Compare at noon so daylight saving changes can't shift the day count.
        local todayNoon = os.time({ year = t.year, month = t.month, day = t.day, hour = 12 })
        local startNoon = os.time({ year = year, month = Config.Month, day = 1, hour = 12 })
        index = math.floor((todayNoon - startNoon) / 86400 + 0.5) + 1
    end

    local phase = index < 1 and 'before' or index > TOTAL_DAYS and 'after' or 'active'
    return year, index, phase
end

local function graceDays()
    return math.max(0, Config.GraceDays or 0)
end

local function getClaims(identifier, year)
    local claimed = {}
    local rows = MySQL.query.await(
        'SELECT `day` FROM `lcrp_halloween_claims` WHERE `identifier` = ? AND `year` = ?',
        { identifier, year }
    ) or {}
    for i = 1, #rows do
        claimed[rows[i].day] = true
    end
    return claimed
end

--- A day is claimable on its own day and for Config.GraceDays days after.
local function dayStatus(day, index, claimed)
    if claimed[day] then return 'claimed' end
    if day > index then return 'locked' end
    if index - day <= graceDays() then return 'available' end
    return 'missed'
end

local function describeRewards(rewards)
    local list = {}
    for i = 1, #rewards do
        local reward = rewards[i]
        if reward.money then
            list[#list + 1] = { name = 'money', label = 'Cash', count = reward.money, money = true }
        else
            local data = itemData(reward.item)
            list[#list + 1] = {
                name = reward.item,
                label = data and data.label or reward.item,
                count = reward.count or 1,
            }
        end
    end
    return list
end

local function buildState(src)
    local identifier = getIdentifier(src)
    if not identifier then return nil end

    local year, index, phase = calendarNow()
    local claimed = getClaims(identifier, year)

    local days = {}
    for day = 1, TOTAL_DAYS do
        days[day] = {
            day = day,
            status = dayStatus(day, index, claimed),
            rewards = describeRewards(Config.Days[day]),
        }
    end

    local nextUnlock
    if not Config.DebugDay and index < TOTAL_DAYS then
        nextUnlock = os.time({ year = year, month = Config.Month, day = math.max(index, 0) + 1, hour = 0 })
    end

    return {
        now = os.time(),
        halloween = os.time({ year = year, month = Config.Month, day = TOTAL_DAYS, hour = 0 }),
        nextUnlock = nextUnlock,
        today = index,
        phase = phase,
        graceDays = graceDays(),
        days = days,
    }
end

local function countAvailable(state)
    local count = 0
    for i = 1, #state.days do
        if state.days[i].status == 'available' then count += 1 end
    end
    return count
end

--- Checks the whole reward fits in the player's inventory before anything is given.
local function canCarryAll(src, rewards)
    local totalWeight = 0
    for i = 1, #rewards do
        local reward = rewards[i]
        if reward.item then
            local count = reward.count or 1
            if not exports.ox_inventory:CanCarryItem(src, reward.item, count) then
                return false
            end
            local data = itemData(reward.item)
            totalWeight += (data and data.weight or 0) * count
        end
    end
    return exports.ox_inventory:CanCarryWeight(src, totalWeight) ~= false
end

local function giveRewards(src, day, rewards)
    for i = 1, #rewards do
        local reward = rewards[i]
        if reward.money then
            exports.qbx_core:AddMoney(src, 'cash', reward.money, ('halloween-calendar-day-%d'):format(day))
        else
            local success, response = exports.ox_inventory:AddItem(src, reward.item, reward.count or 1)
            if not success then
                lib.print.warn(('Failed to give %s x%d to %s for day %d (%s)'):format(
                    reward.item, reward.count or 1, GetPlayerName(src), day, tostring(response)))
            end
        end
    end
end

local function claim(src, day)
    if not exports.qbx_core:GetPlayer(src) then
        return { ok = false, message = 'Your character is not loaded yet.' }
    end

    local identifier = getIdentifier(src)
    if not identifier then
        return { ok = false, message = 'Could not find your identifier.' }
    end

    local year, index = calendarNow()
    local claimed = getClaims(identifier, year)
    local status = dayStatus(day, index, claimed)

    if status ~= 'available' then
        local reasons = {
            claimed = 'You already claimed this treat.',
            locked = 'This treat has not unlocked yet.',
            missed = 'This treat has expired.',
        }
        return { ok = false, message = reasons[status] or 'You cannot claim this day.', state = buildState(src) }
    end

    local rewards = Config.Days[day]
    if not canCarryAll(src, rewards) then
        return { ok = false, message = 'Your pockets are too full. Make some room and try again.' }
    end

    -- The primary key makes this the single source of truth: if two claims race,
    -- only one insert affects a row and only that one hands out rewards.
    local affected = MySQL.update.await(
        'INSERT IGNORE INTO `lcrp_halloween_claims` (`identifier`, `year`, `day`) VALUES (?, ?, ?)',
        { identifier, year, day }
    )
    if not affected or affected < 1 then
        return { ok = false, message = 'You already claimed this treat.', state = buildState(src) }
    end

    giveRewards(src, day, rewards)

    return {
        ok = true,
        message = day == TOTAL_DAYS and 'Happy Halloween! You claimed the grand prize!'
            or ('You claimed the treat for October %d!'):format(day),
        state = buildState(src),
    }
end

---------------------------------------------------------------------------
-- Callbacks
---------------------------------------------------------------------------

lib.callback.register('lcrp_halloween:getState', function(src)
    return buildState(src)
end)

lib.callback.register('lcrp_halloween:claim', function(src, day)
    day = tonumber(day)
    if not day or day % 1 ~= 0 or day < 1 or day > TOTAL_DAYS then
        return { ok = false, message = 'Invalid day.' }
    end
    if busy[src] then
        return { ok = false, message = 'Slow down!' }
    end

    busy[src] = true
    local success, result = pcall(claim, src, day)
    busy[src] = nil

    if not success then
        lib.print.error(('Claim failed for %s (day %d): %s'):format(GetPlayerName(src), day, result))
        return { ok = false, message = 'Something went wrong. Please try again.' }
    end
    return result
end)

---------------------------------------------------------------------------
-- Chat messages
---------------------------------------------------------------------------

--- True while any treat can still be claimed (the month plus the grace period after it).
local function isRunning(index)
    return index >= 1 and index <= TOTAL_DAYS + graceDays()
end

RegisterNetEvent('lcrp_halloween:server:loaded', function()
    local src = source
    if not Config.JoinMessage or welcomed[src] then return end
    welcomed[src] = true

    local _, index = calendarNow()
    if not isRunning(index) then return end

    local state = buildState(src)
    local available = state and countAvailable(state) or 0
    local cmd = '/' .. Config.Command

    if available > 0 then
        local expiring = index > 1 and state.days[index - 1] and state.days[index - 1].status == 'available'
        chat(src, ('You have %d unclaimed Halloween treat%s waiting%s. Type %s to claim.'):format(
            available, available == 1 and '' or 's',
            expiring and ' (yesterday\'s expires at midnight)' or '', cmd))
    elseif index < TOTAL_DAYS then
        chat(src, ('You have claimed every treat so far. Come back tomorrow, or type %s to see the countdown.'):format(cmd))
    end
end)

AddEventHandler('playerDropped', function()
    local src = source
    welcomed[src] = nil
    busy[src] = nil
end)

if Config.Broadcast.enabled then
    CreateThread(function()
        local interval = math.max(1, Config.Broadcast.intervalMinutes) * 60000
        while true do
            Wait(interval)
            local _, index = calendarNow()
            local cmd = '/' .. Config.Command
            if index >= 1 and index < TOTAL_DAYS then
                local left = TOTAL_DAYS - index
                chat(-1, ('%d day%s until Halloween! Type %s to claim today\'s treat. Don\'t wait, treats expire!')
                    :format(left, left == 1 and '' or 's', cmd))
            elseif index == TOTAL_DAYS then
                chat(-1, ('Happy Halloween! The grand prize is waiting for you. Type %s to claim it.'):format(cmd))
            elseif isRunning(index) then
                chat(-1, ('Last chance! The Halloween grand prize can still be claimed with %s.'):format(cmd))
            end
        end
    end)
end

---------------------------------------------------------------------------
-- Startup
---------------------------------------------------------------------------

CreateThread(function()
    MySQL.query.await([[
        CREATE TABLE IF NOT EXISTS `lcrp_halloween_claims` (
            `identifier` VARCHAR(64) NOT NULL,
            `year` SMALLINT UNSIGNED NOT NULL,
            `day` TINYINT UNSIGNED NOT NULL,
            `claimed_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (`identifier`, `year`, `day`)
        )
    ]])

    -- Warn about any configured item ox_inventory does not know about.
    for day = 1, TOTAL_DAYS do
        for _, reward in ipairs(Config.Days[day]) do
            if reward.item and not itemData(reward.item) then
                lib.print.warn(('Day %d reward "%s" does not exist in ox_inventory'):format(day, reward.item))
            end
        end
    end

    if Config.DebugDay then
        lib.print.warn(('%s is running with Config.DebugDay = %d. Turn it off on the live server.')
            :format(RESOURCE, Config.DebugDay))
    end
end)
