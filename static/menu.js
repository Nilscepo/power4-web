// menu.js : stocke les pseudos et le mode puis redirige vers /game
document.addEventListener('DOMContentLoaded', () => {
  const start = document.getElementById('start');
  const p1 = document.getElementById('p1');
  const p2 = document.getElementById('p2');

  start.addEventListener('click', () => {
    const j1 = (p1.value || 'Rouge').trim();
    const j2 = (p2.value || 'Jaune').trim();
    // sauvegarde locale : le jeu pourra lire ces valeurs plus tard
    try { localStorage.setItem('p1', j1); localStorage.setItem('p2', j2); } catch(e) {}
    // rediriger vers la page du jeu
    window.location.href = '/game';
  });
});
