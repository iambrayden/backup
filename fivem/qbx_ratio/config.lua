Config = {}

-- How long (seconds) the HUD stays on screen after a player picks a character.
Config.JoinDisplaySeconds = 60

-- How long (seconds) the HUD stays on screen after typing /ratio.
Config.CommandDisplaySeconds = 15

-- Command name players type in chat (no slash).
Config.Command = 'ratio'

-- How often (seconds) the HUD re-fetches numbers while visible.
Config.RefreshSeconds = 5

-- false = anyone holding the job counts toward that category, on duty or not (current setting).
-- true  = an off-duty cop/medic counts as a civilian.
Config.RequireOnDuty = false

-- How players are put into categories.
-- A player's job is checked against `jobNames` (exact job name from qbx_core/shared/jobs.lua),
-- then against `jobTypes` (the job's `type` field). Anyone who matches nothing is a civilian.
-- jobTypes is left empty so ONLY the listed job names count; jobs not listed are civilian.
Config.Categories = {
    {
        id = 'leo',
        label = 'LEO',
        color = '#3b82f6',
        jobTypes = {},
        jobNames = { 'police', 'bcso', 'sahp', 'lspd' },
    },
    {
        id = 'fire',
        label = 'Fire/EMS',
        color = '#ef4444',
        jobTypes = {},
        jobNames = { 'ambulance', 'safr' },
    },
}

Config.Civilian = {
    id = 'civ',
    label = 'Civilian',
    color = '#22c55e',
}
