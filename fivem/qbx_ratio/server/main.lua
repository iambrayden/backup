-- Build fast lookup sets from the config lists.
local byName, byType = {}, {}
for _, cat in ipairs(Config.Categories) do
    for _, name in ipairs(cat.jobNames or {}) do byName[name] = cat.id end
    for _, jobType in ipairs(cat.jobTypes or {}) do byType[jobType] = cat.id end
end

local CIV = Config.Civilian.id

---@param job table|nil PlayerData.job from qbx_core
---@return string categoryId
local function classify(job)
    if not job then return CIV end

    local category = byName[job.name] or (job.type and byType[job.type])
    if not category then return CIV end

    if Config.RequireOnDuty and not job.onduty then return CIV end

    return category
end

-- Cache briefly so many players opening the HUD at once doesn't loop players repeatedly.
local CACHE_MS = 2000
local cached, cachedAt = nil, 0

local function buildRatio()
    local now = GetGameTimer()
    if cached and now - cachedAt < CACHE_MS then return cached end

    local counts = { [CIV] = 0 }
    for _, cat in ipairs(Config.Categories) do counts[cat.id] = 0 end

    local total = 0
    -- GetQBPlayers only returns players who have loaded a character,
    -- so people sitting in character select / loading screens are not counted.
    for _, player in pairs(exports.qbx_core:GetQBPlayers()) do
        local category = classify(player.PlayerData and player.PlayerData.job)
        counts[category] = (counts[category] or 0) + 1
        total = total + 1
    end

    local rows = {}
    for _, cat in ipairs(Config.Categories) do
        rows[#rows + 1] = { id = cat.id, label = cat.label, color = cat.color, count = counts[cat.id] }
    end
    rows[#rows + 1] = {
        id = CIV,
        label = Config.Civilian.label,
        color = Config.Civilian.color,
        count = counts[CIV],
    }

    cached = { total = total, rows = rows }
    cachedAt = now
    return cached
end

lib.callback.register('qbx_ratio:server:getRatio', function()
    return buildRatio()
end)
