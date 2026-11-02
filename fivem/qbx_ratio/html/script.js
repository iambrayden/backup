const panel = document.getElementById('ratio');
const rowsEl = document.getElementById('rows');
const barEl = document.getElementById('bar');
const totalEl = document.getElementById('total');

function render(data) {
    const total = data.total || 0;
    totalEl.textContent = `${total} online`;

    rowsEl.replaceChildren();
    barEl.replaceChildren();

    for (const row of data.rows || []) {
        const pct = total > 0 ? Math.round((row.count / total) * 100) : 0;

        const seg = document.createElement('span');
        seg.style.background = row.color;
        seg.style.width = total > 0 ? `${(row.count / total) * 100}%` : '0%';
        barEl.appendChild(seg);

        const el = document.createElement('div');
        el.className = 'row';

        const dot = document.createElement('span');
        dot.className = 'dot';
        dot.style.background = row.color;

        const label = document.createElement('span');
        label.className = 'label';
        label.textContent = row.label;

        const count = document.createElement('span');
        count.className = 'count';
        count.textContent = row.count;

        const pctEl = document.createElement('span');
        pctEl.className = 'pct';
        pctEl.textContent = `${pct}%`;

        el.append(dot, label, count, pctEl);
        rowsEl.appendChild(el);
    }
}

window.addEventListener('message', (event) => {
    const msg = event.data || {};
    switch (msg.action) {
        case 'update': render(msg.data || {}); break;
        case 'show': panel.classList.remove('hidden'); break;
        case 'hide': panel.classList.add('hidden'); break;
    }
});
