const RESOURCE = typeof GetParentResourceName === 'function' ? GetParentResourceName() : 'lcrp_halloween';
const IMAGE_PATH = 'nui://ox_inventory/web/images/';
const MONTH = 'October';

const FALLBACK_EMOJI = {
    money: '💵',
    firework1: '🎆', firework2: '🎆', firework3: '🎆', firework4: '🎆',
    WEAPON_FLASHLIGHT: '🔦',
    knife: '🔪',
    fishing_rod: '🎣', fishing_kit: '🧰',
    metal_detector: '📡',
    repair_kit: '🔧',
    bandage: '🩹', gauze: '🩹', firstaid: '⛑️',
};

const el = (id) => document.getElementById(id);
const app = el('app');

let state = null;
let selectedDay = null;
let serverOffset = 0;
let claiming = false;
let refreshing = false;
let ticker = null;

function post(endpoint, data = {}) {
    return fetch(`https://${RESOURCE}/${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json; charset=UTF-8' },
        body: JSON.stringify(data),
    }).then((res) => res.json()).catch(() => null);
}

function serverNow() {
    return Math.floor(Date.now() / 1000) + serverOffset;
}

function pad(n) {
    return String(n).padStart(2, '0');
}

function splitTime(seconds) {
    seconds = Math.max(0, seconds);
    return {
        d: Math.floor(seconds / 86400),
        h: Math.floor((seconds % 86400) / 3600),
        m: Math.floor((seconds % 3600) / 60),
        s: seconds % 60,
    };
}

function fallback(name) {
    return FALLBACK_EMOJI[name] || '🍬';
}

function iconFor(reward, className) {
    const img = document.createElement('img');
    img.className = className;
    img.src = `${IMAGE_PATH}${reward.name}.png`;
    img.alt = '';
    img.onerror = () => {
        const span = document.createElement('span');
        span.className = 'emoji';
        span.textContent = fallback(reward.name);
        img.replaceWith(span);
    };
    return img;
}

function formatCount(reward) {
    return reward.money ? `$${reward.count.toLocaleString()}` : `x${reward.count}`;
}

/* ---------- Countdown ---------- */

function tick() {
    if (!state) return;
    const now = serverNow();
    const main = el('countdown-main');
    const sub = el('countdown-sub');

    const toHalloween = state.halloween - now;
    if (toHalloween <= 0 || state.phase === 'after') {
        main.classList.add('done');
        main.textContent = state.phase === 'after' ? 'See you next Halloween!' : 'Happy Halloween!';
    } else {
        if (main.classList.contains('done')) {
            main.classList.remove('done');
            main.innerHTML = `
                <div class="unit"><span id="cd-days">00</span><small>Days</small></div>
                <div class="unit"><span id="cd-hours">00</span><small>Hours</small></div>
                <div class="unit"><span id="cd-mins">00</span><small>Minutes</small></div>
                <div class="unit"><span id="cd-secs">00</span><small>Seconds</small></div>`;
        }
        const t = splitTime(toHalloween);
        el('cd-days').textContent = pad(t.d);
        el('cd-hours').textContent = pad(t.h);
        el('cd-mins').textContent = pad(t.m);
        el('cd-secs').textContent = pad(t.s);
    }

    if (state.nextUnlock) {
        const remaining = state.nextUnlock - now;
        if (remaining <= 0) {
            sub.textContent = 'A new treat just unlocked!';
            refresh();
        } else {
            const t = splitTime(remaining);
            const clock = `${t.d > 0 ? `${t.d}d ` : ''}${pad(t.h)}:${pad(t.m)}:${pad(t.s)}`;
            const label = state.phase === 'before' ? 'The calendar opens in' : 'Next treat unlocks in';
            sub.innerHTML = `${label} <b>${clock}</b>`;
        }
    } else if (state.phase === 'active') {
        sub.textContent = 'Every treat has been unlocked. Happy haunting!';
    } else if (state.phase === 'after') {
        sub.textContent = 'The Halloween calendar has ended.';
    } else {
        sub.textContent = '';
    }
}

async function refresh() {
    if (refreshing) return;
    refreshing = true;
    const fresh = await post('refresh');
    refreshing = false;
    if (fresh) setState(fresh);
}

/* ---------- Calendar ---------- */

function setState(next) {
    state = next;
    serverOffset = state.now - Math.floor(Date.now() / 1000);

    if (!selectedDay || !state.days[selectedDay - 1]) {
        const firstAvailable = state.days.find((d) => d.status === 'available');
        selectedDay = firstAvailable ? firstAvailable.day : Math.max(1, Math.min(state.today || 1, state.days.length));
    }

    renderGrid();
    renderDetail();
    tick();
}

function renderGrid() {
    const grid = el('grid');
    grid.innerHTML = '';
    const lastDay = state.days.length;

    state.days.forEach((day) => {
        const tile = document.createElement('div');
        const isGrand = day.day === lastDay;
        tile.className = `tile ${day.status}${isGrand ? ' grand' : ''}${day.day === selectedDay ? ' selected' : ''}`;
        tile.onclick = () => {
            selectedDay = day.day;
            renderGrid();
            renderDetail();
        };

        const num = document.createElement('span');
        num.className = 'num';
        num.textContent = day.day;
        tile.appendChild(num);

        const badges = { claimed: '✔', locked: '🔒', missed: '✖' };
        if (badges[day.status]) {
            const badge = document.createElement('span');
            badge.className = 'badge';
            badge.textContent = badges[day.status];
            tile.appendChild(badge);
        }

        if (day.rewards.length) tile.appendChild(iconFor(day.rewards[0], 'icon'));

        if (isGrand) {
            const label = document.createElement('span');
            label.className = 'grand-label';
            label.textContent = 'Halloween Grand Prize';
            tile.appendChild(label);
        }

        grid.appendChild(tile);
    });
}

function renderDetail() {
    const day = state.days[selectedDay - 1];
    if (!day) return;
    const isGrand = day.day === state.days.length;

    el('detail-eyebrow').textContent = isGrand ? 'Grand Prize' : `Day ${day.day}`;
    el('detail-title').textContent = `${MONTH} ${day.day}`;

    const list = el('detail-rewards');
    list.innerHTML = '';
    day.rewards.forEach((reward) => {
        const li = document.createElement('li');
        li.appendChild(iconFor(reward, ''));
        const label = document.createElement('span');
        label.className = 'label';
        label.textContent = reward.label;
        const count = document.createElement('span');
        count.className = 'count';
        count.textContent = formatCount(reward);
        li.append(label, count);
        list.appendChild(li);
    });

    const button = el('claim');
    const note = el('detail-note');
    const texts = {
        available: ['Claim Treat', ''],
        claimed: ['Claimed', 'You already collected this treat.'],
        locked: ['Locked', `Unlocks on ${MONTH} ${day.day}.`],
        missed: ['Missed', 'This treat can no longer be claimed.'],
    };
    const [label, hint] = texts[day.status] || ['Unavailable', ''];
    button.textContent = claiming ? 'Claiming…' : label;
    button.disabled = claiming || day.status !== 'available';
    note.textContent = hint;
}

async function claim() {
    if (claiming || !state) return;
    const day = state.days[selectedDay - 1];
    if (!day || day.status !== 'available') return;

    claiming = true;
    renderDetail();
    const result = await post('claim', { day: day.day });
    claiming = false;

    if (result && result.state) {
        setState(result.state);
    } else {
        renderDetail();
    }
    if (result && !result.ok && result.message) {
        el('detail-note').textContent = result.message;
    }
}

/* ---------- Open / close ---------- */

function open(next) {
    selectedDay = null;
    setState(next);
    app.classList.remove('hidden');
    clearInterval(ticker);
    ticker = setInterval(tick, 1000);
}

function hide() {
    app.classList.add('hidden');
    clearInterval(ticker);
    ticker = null;
}

function close() {
    hide();
    post('close');
}

el('close').addEventListener('click', close);
el('claim').addEventListener('click', claim);

document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !app.classList.contains('hidden')) close();
});

window.addEventListener('message', (event) => {
    const data = event.data || {};
    if (data.action === 'open') open(data.state);
    if (data.action === 'close') hide();
});
