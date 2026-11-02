local hideAt = 0
local visible = false
local refreshNow = false

local function pushRatio()
    local data = lib.callback.await('qbx_ratio:server:getRatio', false)
    if data then
        SendNUIMessage({ action = 'update', data = data })
    end
end

--- Show the HUD for `seconds`, extending the timer if it's already visible.
local function showFor(seconds)
    local target = GetGameTimer() + seconds * 1000
    if target > hideAt then hideAt = target end

    if visible then
        refreshNow = true
        return
    end

    visible = true
    CreateThread(function()
        pushRatio()
        SendNUIMessage({ action = 'show' })

        local nextRefresh = GetGameTimer() + Config.RefreshSeconds * 1000
        while GetGameTimer() < hideAt do
            Wait(250)
            if refreshNow or GetGameTimer() >= nextRefresh then
                refreshNow = false
                pushRatio()
                nextRefresh = GetGameTimer() + Config.RefreshSeconds * 1000
            end
        end

        SendNUIMessage({ action = 'hide' })
        visible = false
    end)
end

local function hide()
    hideAt = 0 -- the display loop exits on its next tick
end

-- Fired by qbx_core once the player has picked/spawned a character.
RegisterNetEvent('QBCore:Client:OnPlayerLoaded', function()
    showFor(Config.JoinDisplaySeconds)
end)

RegisterNetEvent('QBCore:Client:OnPlayerUnload', hide)

RegisterCommand(Config.Command, function()
    if not LocalPlayer.state.isLoggedIn then return end
    showFor(Config.CommandDisplaySeconds)
end, false)

CreateThread(function()
    TriggerEvent('chat:addSuggestion', '/' .. Config.Command, 'Show the LEO / Fire-EMS / Civilian ratio')
end)
