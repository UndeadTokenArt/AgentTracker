(() => {
  const body = document.body;
  const ctx = { code: body.dataset.code || '', isDM: String(body.dataset.isdm) === 'true' };
  const list = document.getElementById('list');
  const roundEl = document.getElementById('round');
  const turnEl = document.getElementById('turn');
  const nextBtn = document.getElementById('nextBtn');

  const ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws/${ctx.code}`);

  ws.addEventListener('message', (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.type === 'state') {
      const { round, turn, entries, dmUid } = msg.data;
      roundEl.textContent = round;
      turnEl.textContent = entries.length ? (turn + 1) : 0;
      render(entries, turn);
      if (!ctx.isDM) {
        document.querySelector('small.text-muted')?.classList.add('d-none');
      }
    }
  });

  function render(entries, turn) {
    list.innerHTML = '';
    entries.forEach((e, i) => {
      const li = document.createElement('li');
      li.className = `list-group-item bg-dark text-light d-flex justify-content-between align-items-center entity ${e.type}`;
      li.dataset.id = e.id;
      const left = document.createElement('div');
      left.className = 'd-flex flex-column';
      left.innerHTML = `<div class="fw-semibold">${escapeHtml(e.name)}</div>
                        <div class="small text-secondary">Init ${e.initiative}${e.bonus?` (b+${e.bonus})`:''}</div>`;
      li.appendChild(left);
      const right = document.createElement('div');
      right.className = 'd-flex align-items-center gap-2';
      if (e.type === 'monster' && ctx.isDM) {
        const hpWrap = document.createElement('div');
        hpWrap.style.minWidth = '120px'; hpWrap.className = 'text-end';
        hpWrap.innerHTML = `<div class="small">HP ${e.hp}/${e.maxHp}</div>
                            <div class="hpbar"><div class="inner" style="width:${pct(e.hp,e.maxHp)}%"></div></div>`;
        right.appendChild(hpWrap);
        const dmgBtn = document.createElement('button');
        dmgBtn.className = 'btn btn-sm btn-outline-danger';
        dmgBtn.textContent = 'Damage';
        dmgBtn.onclick = () => {
          const v = parseInt(prompt('Damage amount?')||'0',10);
          if (!Number.isFinite(v) || v <= 0) return; 
          wsSend('damage', { id: e.id, dmg: v });
        };
        right.appendChild(dmgBtn);
      }
      li.appendChild(right);
      if (i === turn) li.classList.add('active');
      list.appendChild(li);
    });
  }

  function wsSend(type, data) {
    ws.readyState === 1 && ws.send(JSON.stringify({ type, data }));
  }

  function escapeHtml(s){return s.replace(/[&<>"]+/g, c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));}
  function pct(a,b){ if(!b) return 0; return Math.max(0,Math.min(100, Math.round(a*100/b))); }

  // Forms
  const pf = document.getElementById('playerForm');
  pf?.addEventListener('submit', (e) => {
    e.preventDefault();
    const fd = new FormData(pf);
    const name = (fd.get('name')||'').toString();
    const initiative = Math.max(0, parseInt(fd.get('initiative')||'0',10));
    const bonus = parseInt(fd.get('bonus')||'0',10) || 0;
    wsSend('addPlayer', { name, initiative, bonus });
    pf.reset();
  });
  document.getElementById('rollBtn')?.addEventListener('click', ()=>{
    const fd = new FormData(pf);
    const name = (fd.get('name')||'').toString();
    const bonus = parseInt(fd.get('bonus')||'0',10) || 0;
    wsSend('addPlayerRoll', { name, bonus });
    pf.reset();
  });

  const mf = document.getElementById('monsterForm');
  mf?.addEventListener('submit', (e) => {
    e.preventDefault();
    const fd = new FormData(mf);
    const name = (fd.get('name')||'').toString();
    const hp = parseInt(fd.get('hp')||'0',10) || 0;
    const initiative = Math.max(0, parseInt(fd.get('initiative')||'0',10));
    const bonus = parseInt(fd.get('bonus')||'0',10) || 0;
    wsSend('addMonster', { name, hp, initiative, bonus });
    mf.reset();
  });

  nextBtn?.addEventListener('click', ()=> wsSend('next', {}));

  // Reset button (DM only)
  const resetBtn = document.getElementById('resetBtn');
  resetBtn?.addEventListener('click', () => {
    if (confirm('Are you sure you want to reset the initiative? This will remove all players and monsters.')) {
      wsSend('reset', {});
    }
  });

  // Drag and drop for DM
  if (ctx.isDM) {
    new Sortable(list, {
      animation: 150,
      onEnd: () => {
        const order = Array.from(list.children).map(li => li.dataset.id);
        wsSend('reorder', { order });
      }
    });
  }
})();
