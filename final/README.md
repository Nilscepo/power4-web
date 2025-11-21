# Puissance4BKNE

Description
- Projet web implémentant le jeu Puissance 4.
- Backend en Go, frontend en HTML/JS/CSS. L'application sert l'interface web et la logique du jeu pour jouer en local via un navigateur.

Technologies
- Langage : Go (module `go.mod`)
- Frontend : HTML, CSS, JavaScript (fichiers sous `static/`)
- Compatible Windows / PowerShell pour le développement local

Structure du projet (répertoire `final`)
- `main.go` : point d'entrée du serveur.
- `_liaisonFront.go` : code pour lier le frontend au backend (routes/handlers).
- `_logiquePower.go` : logique du jeu (règles, états).
- `_startGame.go` : initialisation de partie / gestion du flow.
- `index.html`, `menu.html` : pages frontend.
- `static/` : ressources statiques : `game.js`, `menu.js`, `power4.css`.

Comment lancer (développement local)
- Ouvrir PowerShell dans le dossier `final`.
- Lancer le serveur :
  ```powershell
  go run .
  ```
  ou git bash go run main.go

- Ouvrir le navigateur à l'URL indiquée par le serveur (généralement `http://localhost:8080` ou l'URL affichée dans la console).

Utilisation
- Accéder à la page d'accueil, choisir une partie depuis le menu et jouer directement dans l'interface.
- `game.js` gère l'interaction et communique avec le backend pour valider coups et états.

Notes pour les développeurs
- Le code est structuré pour séparer logique et liaison frontend.

Projet de 

Baptiste Ancelin
Rio Killian
Bidar Elias
Cepo Nils
