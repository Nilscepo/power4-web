// JS minimaliste : gère clics, envoie /play, met à jour DOM et anime chute

async function getState() {
  const res = await fetch('/state');
  return res.json();
}

// dernier état côté client pour ne montrer les popups qu'une seule fois
let _lastEtat = { vainqueur: 0, egalite: false };

function formatTime(s) {
  const m = Math.floor(s / 60).toString().padStart(2, '0');
  const sec = (s % 60).toString().padStart(2, '0');
  return `${m}:${sec}`;
}

function updateDOM(etat) {
  // mettre à jour les cellules
  for (let r = 0; r < 6; r++) {
    for (let c = 0; c < 7; c++) {
      // sélectionner la cellule précisément par classe (évite problèmes d'ordre DOM)
      const selector = `.ligne_${r+1} .A${c+1}_creux`;
      const cell = document.querySelector(selector);
      if (!cell) continue;
      cell.classList.remove('rouge', 'jaune');
      const v = etat.plateau[r][c];
      if (v === 1) cell.classList.add('rouge');
      if (v === 2) cell.classList.add('jaune');
    }
  }
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
  // row: 0..5 top->bottom; col: 0..6
  const selector = `.ligne_${row+1} .A${col+1}_creux`;
  const cell = document.querySelector(selector);
  if (!cell) return;
  const cls = player === 1 ? 'rouge' : 'jaune';
  cell.classList.remove('rouge','jaune');
  cell.style.transform = 'translateY(-200px) scale(0.6)';
  cell.classList.add('jeton-anim');
  // donner la couleur avant l'animation pour la voir tomber
  cell.classList.add(cls);
  // forcer reflow
  cell.offsetHeight;
  // animer vers la position
  cell.style.transform = '';
  setTimeout(() => {
    cell.classList.remove('jeton-anim');
    cell.style.transform = '';
  }, 600);
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
  for (let i = 1; i <= 7; i++) {
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
      const colWidth = rect.width / 7;
      let col = Math.floor(x / colWidth);
      if (col < 0) col = 0;
      if (col > 6) col = 6;
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
