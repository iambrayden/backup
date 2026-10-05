local isOpen = false

local function close()
    if not isOpen then return end
    isOpen = false
    SetNuiFocus(false, false)
    SendNUIMessage({ action = 'close' })
end

local function open()
    if isOpen then return end

    local state = lib.callback.await('lcrp_halloween:getState', false)
    if not state then
        lib.notify({ title = 'Halloween', description = 'Could not load the calendar.', type = 'error' })
        return
    end

    isOpen = true
    SetNuiFocus(true, true)
    SendNUIMessage({ action = 'open', state = state })
end

RegisterCommand(Config.Command, open, false)

CreateThread(function()
    TriggerEvent('chat:addSuggestion', '/' .. Config.Command, 'Open the Halloween countdown & daily rewards')
end)

RegisterNUICallback('close', function(_, cb)
    close()
    cb(1)
end)

RegisterNUICallback('refresh', function(_, cb)
    cb(lib.callback.await('lcrp_halloween:getState', false) or false)
end)

RegisterNUICallback('claim', function(data, cb)
    local result = lib.callback.await('lcrp_halloween:claim', false, data.day)
        or { ok = false, message = 'No response from the server.' }

    lib.notify({
        title = 'Halloween',
        description = result.message,
        type = result.ok and 'success' or 'error',
    })

    cb(result)
end)

RegisterNetEvent('QBCore:Client:OnPlayerLoaded', function()
    TriggerServerEvent('lcrp_halloween:server:loaded')
end)

AddEventHandler('onResourceStop', function(resource)
    if resource == GetCurrentResourceName() and isOpen then
        SetNuiFocus(false, false)
    end
end)
