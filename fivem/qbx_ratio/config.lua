Config = {}

-- How long (seconds) the HUD stays on screen after a player picks a character.
Config.JoinDisplaySeconds = 60

-- How long (seconds) the HUD stays on screen after typing /ratio.
Config.CommandDisplaySeconds = 15

-- Command name players type in chat (no slash).
Config.Command = 'ratio'

-- How often (seconds) the HUD re-fetches numbers while visible.
Config.RefreshSeconds = 5

-- true  = an off-duty cop/medic counts as a civilian (recommended: reflects who is actually working).
-- false = anyone holding the job counts toward that category, on duty or not.
Config.RequireOnDuty = true

-- How players are put into categories.
-- A player's job is checked against `jobNames` first (exact job name from qbx_core/shared/jobs.lua),
-- then against `jobTypes` (the `type` field on the job, e.g. Qbox ships police as type 'leo'
-- and ambulance as type 'ems'). Anyone who matches nothing is a civilian.
-- Add your server's custom jobs (bcso, sasp, safd, etc.) here.
Config.Categories = {
    {
        id = 'leo',
        label = 'LEO',
        color = '#3b82f6',
        jobTypes = { 'leo' },
        jobNames = { 'police', 'lspd', 'bcso', 'sasp', 'sheriff', 'statepolice' },
    },
    {
        id = 'fire',
        label = 'Fire/EMS',
        color = '#ef4444',
        jobTypes = { 'ems', 'fire' },
        jobNames = { 'ambulance', 'ems', 'fire', 'safd', 'lsfd' },
    },
}

Config.Civilian = {
    id = 'civ',
    label = 'Civilian',
    color = '#22c55e',
}
