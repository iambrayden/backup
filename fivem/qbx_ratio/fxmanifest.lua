fx_version 'cerulean'
game 'gta5'
lua54 'yes'

name 'qbx_ratio'
description 'Shows the LEO / Fire-EMS / Civilian player ratio (/ratio + on-join HUD)'
version '1.0.0'

shared_scripts {
    '@ox_lib/init.lua',
    'config.lua',
}

client_script 'client/main.lua'
server_script 'server/main.lua'

ui_page 'html/index.html'

files {
    'html/index.html',
    'html/style.css',
    'html/script.js',
}

dependencies {
    'qbx_core',
    'ox_lib',
}
