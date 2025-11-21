// JS minimaliste : gère clics, envoie /play, met à jour DOM et anime chute

async function getState() {
  const res = await fetch('/state');
  return res.json();
}

// dernier état côté client pour ne montrer les popups qu'une seule fois
let _lastEtat = { vainqueur: 0, egalite: false };
// grille actuelle
let GRID_ROWS = 6;
let GRID_COLS = 7;

function formatTime(s) {
  const m = Math.floor(s / 60).toString().padStart(2, '0');
  const sec = (s % 60).toString().padStart(2, '0');
  return `${m}:${sec}`;
}

function updateDOM(etat) {
  // dessiner les jetons en overlay (supporte n'importe quelle taille de grille)
  renderTokens(etat);
  // timers
  const tX = document.getElementById('timer_X');
  const tO = document.getElementById('timer_O');
  if (tX) tX.textContent = formatTime(etat.timers[1] || 0);
  if (tO) tO.textContent = formatTime(etat.timers[2] || 0);
  // afficher gagnant
  if (etat.vainqueur === 1) {
    const w = document.querySelector('.win_rouge');
    if (w) w.style.visibility = 'visible';
    // popup victoire (une seule fois)
    if (!_lastEtat.vainqueur) {
      const name = document.getElementById('name_X')?.textContent || 'Rouge';
      setTimeout(()=> alert(name + ' gagne !'), 100);
    }
  } else {
    const w = document.querySelector('.win_rouge');
    if (w) w.style.visibility = 'hidden';
  }
  if (etat.vainqueur === 2) {
    const w = document.querySelector('.win_jaune');
    if (w) w.style.visibility = 'visible';
  } else {
    const w = document.querySelector('.win_jaune');
    if (w) w.style.visibility = 'hidden';
  }
  // égalité -> popup unique
  if (etat.egalite && !_lastEtat.egalite) {
    setTimeout(()=> alert('Match nul !'), 100);
  }

  // mettre à jour dernier état local
  _lastEtat.vainqueur = etat.vainqueur || 0;
  _lastEtat.egalite = !!etat.egalite;
}

// renderTokens crée/met à jour une couche d'overlay avec les jetons
function renderTokens(etat) {
  const grille = document.querySelector('.grille');
  if (!grille || !etat || !etat.plateau) return;
  const rows = etat.plateau.length || 6;
  const cols = (etat.plateau[0] || []).length || 7;
  GRID_ROWS = rows;
  GRID_COLS = cols;
  let overlay = document.querySelector('.overlay-jetons');
  if (!overlay) {
    overlay = document.createElement('div');
    overlay.className = 'overlay-jetons';
    grille.appendChild(overlay);
  }
  // clear existing
  overlay.innerHTML = '';
  const rect = grille.getBoundingClientRect();
  const cellW = rect.width / cols;
  const cellH = rect.height / rows;
  const size = Math.min(cellW, cellH) * 0.8;
  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < cols; c++) {
      const v = etat.plateau[r][c];
      if (!v) continue;
      const el = document.createElement('div');
      el.className = 'overlay-jeton ' + (v === 1 ? 'rouge' : 'jaune');
      el.style.width = size + 'px';
      el.style.height = size + 'px';
      // position relative to grille
      const left = c * cellW + (cellW - size) / 2;
      const top = r * cellH + (cellH - size) / 2;
      el.style.left = left + 'px';
      el.style.top = top + 'px';
      overlay.appendChild(el);
    }
  }
}

// lire les pseudos dans localStorage et les afficher
function loadAndShowNames(){
  try{
    const p1 = localStorage.getItem('p1') || 'Rouge';
    const p2 = localStorage.getItem('p2') || 'Jaune';
    const n1 = document.getElementById('name_X');
    const n2 = document.getElementById('name_O');
    if(n1) n1.textContent = p1;
    if(n2) n2.textContent = p2;
  }catch(e){/* ignore */}
}

function animateDrop(row, col, player) {
  // create an overlay token that drops into place
  const grille = document.querySelector('.grille');
  if (!grille) return;
  const rect = grille.getBoundingClientRect();
  const overlay = document.querySelector('.overlay-jetons') || (() => {
    const o = document.createElement('div'); o.className = 'overlay-jetons'; grille.appendChild(o); return o;
  })();
  const rows = GRID_ROWS || 6;
  const cols = GRID_COLS || 7;
  const cellW = rect.width / cols;
  const cellH = rect.height / rows;
  const size = Math.min(cellW, cellH) * 0.8;
  const left = col * cellW + (cellW - size) / 2;
  const targetTop = row * cellH + (cellH - size) / 2;

  const el = document.createElement('div');
  el.className = 'overlay-jeton ' + (player === 1 ? 'rouge' : 'jaune');
  el.style.width = size + 'px';
  el.style.height = size + 'px';
  // start above the grille
  el.style.left = left + 'px';
  el.style.top = (-size - 20) + 'px';
  overlay.appendChild(el);
  // force reflow
  el.offsetHeight;
  // animate to target
  el.style.transform = `translateY(${targetTop + size + 20}px)`;
  // remove transform after animation and leave the final element in place
  setTimeout(()=>{
    el.style.transform = '';
    el.style.top = targetTop + 'px';
    // optionally cleanup old overlays then refresh from server state
    setTimeout(()=> getState().then(updateDOM), 200);
  }, 500);
}

async function playCol(c) {
  try {
    disableButtons(true);
    const res = await fetch('/play', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({col: c})
    });
    if (!res.ok) {
      const data = await res.json().catch(()=>({}));
      console.error('erreur', data);
      disableButtons(false);
      return;
    }
    const etat = await res.json();
    // animer la dernière chute
    if (etat.dernier_row >= 0) {
      animateDrop(etat.dernier_row, etat.dernier_col, (etat.vainqueur? etat.vainqueur : (etat.courant===1?2:1)) );
      // on remet à jour l'ensemble après un court délai
      setTimeout(()=> getState().then(updateDOM), 300);
    } else {
      updateDOM(etat);
    }
    disableButtons(false);
  } catch(e) {
    console.error(e);
    disableButtons(false);
  }
}

function disableButtons(dis) {
  for (let i = 1; i <= GRID_COLS; i++) {
    const b = document.querySelector('.button_' + i);
    if (b) b.disabled = dis;
  }
}

function attachHandlers() {
  for (let i = 1; i <= 7; i++) {
    const b = document.querySelector('.button_' + i);
    if (!b) continue;
    ((col)=>{
      b.addEventListener('click', (ev)=>{ ev.stopPropagation(); playCol(col); });
    })(i-1);
  }
  // clic sur la grille : calculer la colonne à partir de la position X du clic
  const grille = document.querySelector('.grille');
  if (grille) {
    grille.addEventListener('click', (ev) => {
      const rect = grille.getBoundingClientRect();
      const x = ev.clientX - rect.left; // position relative à la grille
      const colWidth = rect.width / GRID_COLS;
      let col = Math.floor(x / colWidth);
      if (col < 0) col = 0;
      if (col > GRID_COLS - 1) col = GRID_COLS - 1;
      playCol(col);
    });
  }
  // reset au double-clic sur head (facultatif)
  const head = document.querySelector('.head');
  if (head) head.addEventListener('dblclick', async ()=>{
    await fetch('/reset', {method:'POST'});
    getState().then(updateDOM);
  });

  // bouton retour au menu
  const back = document.getElementById('backMenu');
  if (back) {
    back.addEventListener('click', async ()=>{
      // réinitialiser la partie côté serveur puis retourner au menu
      try { await fetch('/reset', {method:'POST'}); } catch(e) { /* ignore */ }
      window.location.href = '/';
    });
  }
}

// initialisation
window.addEventListener('DOMContentLoaded', () => {
  attachHandlers();
  loadAndShowNames();
  getState().then(updateDOM);
  // poll léger pour timers (1s)
  setInterval(()=> getState().then(updateDOM), 1000);
});
