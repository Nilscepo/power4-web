// Demande l'état du jeu au serveur et retourne en save
async function getState() {
    const res = await fetch('/state'); // Demande au serveur l'état du jeu
    return res.json(); // Retourne une save JSON
}

// On garde en mémoire le dernier résultat
let _lastEtat = { vainqueur: 0, egalite: false };

// Transforme l'affichage pour afficher le timer
function formatTime(s) {
    const m = Math.floor(s / 60).toString().padStart(2, '0');
    const sec = (s % 60).toString().padStart(2, '0');
    return `${m}:${sec}`; // résultat 
}

// Met à jour ce qu'on voit à l'écran selon l'état du jeu
function updateDOM(etat) {
    // On parcourt toutes les cases du plateau
    for (let r = 0; r < 6; r++) {
    for (let c = 0; c < 7; c++) {
        const selector = `.ligne_${r+1} .A${c+1}_creux`; // trouve la case correspondante
        const cell = document.querySelector(selector); 
        if (!cell) continue; // si on ne trouve pas la case, on passe
        cell.classList.remove('rouge', 'jaune'); // on enlève la couleur précédente
        const v = etat.plateau[r][c]; // valeur de la case
        if (v === 1) cell.classList.add('rouge'); // si joueur 1, rouge
        if (v === 2) cell.classList.add('jaune'); // si joueur 2, jaune
        }
    }

    // Met à jour les timers des joueurs
    const tX = document.getElementById('timer_X');
    const tO = document.getElementById('timer_O');
    if (tX) tX.textContent = formatTime(etat.timers[1] || 0);
    if (tO) tO.textContent = formatTime(etat.timers[2] || 0);

    // Affiche le message de victoire pour le joueur rouge
    if (etat.vainqueur === 1) {
        const w = document.querySelector('.win_rouge');
        if (w) w.style.visibility = 'visible'; // montre le message
        // popup d'alerte seulement si pas déjà montré
        if (!_lastEtat.vainqueur) {
        const name = document.getElementById('name_X')?.textContent || 'Rouge';
        setTimeout(()=> alert(name + ' gagne !'), 100);
        }
    } else {
        const w = document.querySelector('.win_rouge');
        if (w) w.style.visibility = 'hidden'; // sinon on cache le message
    }

    // Affiche le message de victoire pour le joueur jaune
    if (etat.vainqueur === 2) {
        const w = document.querySelector('.win_jaune');
        if (w) w.style.visibility = 'visible';
    } else {
        const w = document.querySelector('.win_jaune');
        if (w) w.style.visibility = 'hidden';
    }

    // Si égalité, on montre un popup une seule fois
    if (etat.egalite && !_lastEtat.egalite) {
        setTimeout(()=> alert('Match nul !'), 100);
    }

    // On enregistre l'état actuel pour ne pas répéter les popups
    _lastEtat.vainqueur = etat.vainqueur || 0;
    _lastEtat.egalite = !!etat.egalite;
}

// Affiche les noms des joueurs depuis le navigateur
function loadAndShowNames(){
    try{
        const p1 = localStorage.getItem('p1') || 'Rouge'; // joueur 1
        const p2 = localStorage.getItem('p2') || 'Jaune'; // joueur 2
        const n1 = document.getElementById('name_X');
        const n2 = document.getElementById('name_O');
        if(n1) n1.textContent = p1; // affiche joueur 1
        if(n2) n2.textContent = p2; // affiche joueur 2
    }catch(e){/* ignore erreur si navigateur bloque stockage */}
}

// Anime la chute d'un jeton
function animateDrop(row, col, player) {
    const selector = `.ligne_${row+1} .A${col+1}_creux`;
    const cell = document.querySelector(selector);
    if (!cell) return;
    const cls = player === 1 ? 'rouge' : 'jaune'; // couleur du jeton
    cell.classList.remove('rouge','jaune'); // enlever ancienne couleur
    cell.style.transform = 'translateY(-200px) scale(0.6)'; // départ haut
    cell.classList.add('jeton-anim'); 
    cell.classList.add(cls); // appliquer couleur
    cell.offsetHeight; // force le navigateur à appliquer l'animation
    cell.style.transform = ''; // fait descendre le jeton
    setTimeout(() => {
        cell.classList.remove('jeton-anim'); // animation terminée
        cell.style.transform = '';
    }, 600);
}

// Joue dans une colonne
async function playCol(c) {
    try {
        disableButtons(true); // empêche de cliquer pendant l'envoi
        const res = await fetch('/play', {
        method: 'POST',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify({col: c}) // on envoie la colonne au serveur
        });
        if (!res.ok) {
        const data = await res.json().catch(()=>({}));
        console.error('erreur', data);
        disableButtons(false);
        return;
        }
        const etat = await res.json();
        if (etat.dernier_row >= 0) {
        // animer le dernier jeton
        animateDrop(etat.dernier_row, etat.dernier_col, (etat.vainqueur? etat.vainqueur : (etat.courant===1?2:1)) );
        setTimeout(()=> getState().then(updateDOM), 300); // mettre à jour après chute
        } else {
        updateDOM(etat);
        }
        disableButtons(false); // réactive les boutons
    } catch(e) {
        console.error(e);
        disableButtons(false);
    }
}

// Active ou désactive les boutons des colonnes
function disableButtons(dis) {
    for (let i = 1; i <= 7; i++) {
        const b = document.querySelector('.button_' + i);
        if (b) b.disabled = dis;
    }
}

// Attache tous les événements : clics, double-clic, menu
function attachHandlers() {
    for (let i = 1; i <= 7; i++) {
        const b = document.querySelector('.button_' + i);
        if (!b) continue;
        ((col)=>{
        b.addEventListener('click', (ev)=>{ ev.stopPropagation(); playCol(col); });
        })(i-1);
    }

    const grille = document.querySelector('.grille');
    if (grille) {
        grille.addEventListener('click', (ev) => {
            const rect = grille.getBoundingClientRect();
            const x = ev.clientX - rect.left; // position du clic
            let col = Math.floor(x / (rect.width / 7)); // quelle colonne
            if (col < 0) col = 0;
            if (col > 6) col = 6;
            playCol(col);
        });
    }

    // double-clic sur la tête pour reset
    const head = document.querySelector('.head');
    if (head) head.addEventListener('dblclick', async ()=>{
        await fetch('/reset', {method:'POST'});
        getState().then(updateDOM);
    });

    // bouton retour au menu
    const back = document.getElementById('backMenu');
    if (back) {
        back.addEventListener('click', async ()=>{
        try { await fetch('/reset', {method:'POST'}); } catch(e) { }
        window.location.href = '/'; // retourne au menu
        });
    }
}

// Quand la page est chargée
window.addEventListener('DOMContentLoaded', () => {
    attachHandlers(); // active clics
    loadAndShowNames(); // affiche noms
    getState().then(updateDOM); // affiche état initial
    setInterval(()=> getState().then(updateDOM), 1000); // met à jour les timers chaque seconde
});
