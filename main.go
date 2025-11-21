package main

import (
	"encoding/json" // Pour encoder/décoder du JSON
	"fmt"
	"net/http"
	"path/filepath" // Pour gérer les chemins de fichiers
	"sync"          // Pour synchronisation via mutex (lock/unlock)
	"time"          // Pour gérer le temps, notamment les timers
)

// création de var. lock pour protéger les variables partagées
var mu sync.Mutex

// Variables globales représentant l'état du jeu
var (
	plateau    [][]int // le plateau, matrice de rows x cols contenant 0, 1, ou 2
	rows       = 6
	cols       = 7
	connectN   = 4
	courant    = 1
	vainqueur  = 0
	timers     = map[int]int{1: 180, 2: 180} // chronomètres pour chaque joueur (en sec.)
	egalite    = false
	dernierRow = -1 // dernière ligne où un jeton a été posé
	dernierCol = -1
)

func main() {
	nouveauPlateau() // initialise le plateau au démarrage

	// goroutine (permet la continuité du prog.) qui gère les timers (décrémentation chaque seconde)
	go func() {
		ticker := time.NewTicker(1 * time.Second) // un tick chaque seconde
		for range ticker.C {
			mu.Lock()                       // verrouille pour accès concurrent sécurisé
			if vainqueur == 0 && !egalite { // décrément si pas de vainqueur ni égalité
				if timers[courant] > 0 {
					timers[courant]-- // on enlève 1 seconde au joueur courant
					if timers[courant] <= 0 {
						egalite = true
					}
				}
			}
			mu.Unlock()
		}
	}()

	http.HandleFunc("/", handleIndex) // route pour la page menu

	// sert les fichiers statiques (CSS, JS, images)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/state", handleState) // route qui renvoie l'état du jeu en JSON
	http.HandleFunc("/play", handlePlay)   // route pour jouer un coup
	http.HandleFunc("/reset", handleReset) // route pour réinitialiser la partie
	http.HandleFunc("/game", handleGame)   // page principale du jeu

	addr := ":8080"
	fmt.Println("Serveur démarré sur http://localhost" + addr)

	// démarrage du serveur HTTP sur le port 8080
	if err := http.ListenAndServe(addr, nil); err != nil {
		// si erreur, on essaie le port 8081
		fmt.Println("Erreur ListenAndServe:", err)
		fmt.Println("Tentative sur le port :8081...")
		if err2 := http.ListenAndServe(":8081", nil); err2 != nil {
			fmt.Println("Échec ListenAndServe sur :8081:", err2)
		}
	}
}

// nouveauPlateau réinitialise le jeu
func nouveauPlateau() {
	mu.Lock()         // verrouillage
	defer mu.Unlock() // déverrouille automatiquement quand l'action est finie

	// création d’un plateau vide lignes x colonnes
	plateau = make([][]int, rows) // slice de slice d'entier (dynamique)
	for r := 0; r < rows; r++ {
		plateau[r] = make([]int, cols)
		for c := 0; c < cols; c++ { //--> '1/2' si pion et supp '0'
			plateau[r][c] = 0 // case vide | sert pour vider le plateau
		}
	}

	// réinitialisation des variables de partie "rematch"
	courant = 1
	vainqueur = 0
	timers[1] = 180
	timers[2] = 180
	egalite = false
	dernierRow = -1
	dernierCol = -1
}

// handleIndex sert la page du menu
func handleIndex(w http.ResponseWriter, r *http.Request) { // répond aux requêtes HTTP
	p, _ := filepath.Abs("./menu.html")
	http.ServeFile(w, r, p) // envoie menu.html au client
}

// handleGame sert la page du jeu (index.html)
func handleGame(w http.ResponseWriter, r *http.Request) { // appelée quand le joueur clique sur "Jouer" dans le menu.
	p, _ := filepath.Abs("./index.html")
	http.ServeFile(w, r, p)
}

// Etat représente les données envoyées en JSON au client
type Etat struct {
	Plateau    [][]int     `json:"plateau"`
	Courant    int         `json:"courant"`
	Vainqueur  int         `json:"vainqueur"`
	Timers     map[int]int `json:"timers"`
	DernierRow int         `json:"dernier_row"`
	DernierCol int         `json:"dernier_col"`
	Egalite    bool        `json:"egalite"`
}

// crée copie sécurisée du plateau pour éviter modification concurrente = bug
func copyPlateau() [][]int {
	p := make([][]int, rows) // p = copie du plateau
	for r := 0; r < rows; r++ {
		row := make([]int, cols)
		copy(row, plateau[r]) // copie des éléments
		p[r] = row
	}
	return p
}

// convertir un objet Go → envoyer en JSON au navigateur || renvoye l’état du jeu au front-end
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v) //ignorer uen erreur potentielle
}

// utilitaire pour envoyer une erreur JSON
func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"Non non c'est pas pssible ça": msg})
}

// handleState : renvoie l’état du jeu au client (Maj)
func handleState(w http.ResponseWriter, r *http.Request) { // le front demande l’état du jeu via cette route
	mu.Lock()
	etat := Etat{
		Plateau:    copyPlateau(),
		Courant:    courant,
		Vainqueur:  vainqueur,
		Timers:     map[int]int{1: timers[1], 2: timers[2]},
		DernierRow: dernierRow,
		DernierCol: dernierCol,
		Egalite:    egalite,
	}
	mu.Unlock()
	writeJSON(w, etat)
}

// représente une requête du front-end d'une case cliquer
type PlayReq struct {
	Col int `json:"col"` //col = jouer en front
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	// Vérifie que la méthode HTTP utilisée est POST et pas GET ex
	if r.Method != "POST" { //Envoyer des données / modifier le serveur
		writeError(w, http.StatusMethodNotAllowed, "méthode non autorisée")
		return // arrêt de la fonction ici si ce n'est pas POST
	}

	// Décode la requête JSON envoyée par le front-end pour savoir quelle colonne jouer
	var req PlayReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}
	col := req.Col // colonne choisie par le joueur

	mu.Lock() //éviter les conflits

	// Vérifie si la partie est déjà terminée
	if vainqueur != 0 {
		mu.Unlock()
		writeError(w, http.StatusConflict, "jeu termine")
		return
	}

	// Vérifie si la colonne choisie est valide
	if col < 0 || col >= cols {
		mu.Unlock()
		writeError(w, http.StatusBadRequest, "colonne invalide")
		return
	}

	// Vérifie si la colonne est déjà pleine
	if colonnePleine(col) {
		mu.Unlock()
		writeError(w, http.StatusConflict, "colonne pleine")
		return
	}

	// Place le jeton dans la colonne
	ligne, err := placerJeton(col, courant)
	if err != nil {
		mu.Unlock()
		writeError(w, http.StatusInternalServerError, "impossible de placer")
		return
	}

	// Met à jour les coordonnées du dernier jeton posé
	dernierRow = ligne
	dernierCol = col

	// Vérifie si le joueur courant a gagné
	if verifierVictoire() {
		vainqueur = courant
	} else if isFull() { // et/ou vérifie si le plateau est plein → égalité
		egalite = true
	} else {
		// Sinon, change de joueur pour le tour suivant
		if courant == 1 {
			courant = 2
		} else {
			courant = 1
		}
	}

	// Crée l'état du jeu à renvoyer au front-end
	etat := Etat{
		Plateau:    copyPlateau(),
		Courant:    courant,
		Vainqueur:  vainqueur,
		Timers:     map[int]int{1: timers[1], 2: timers[2]},
		DernierRow: dernierRow,
		DernierCol: dernierCol,
		Egalite:    egalite,
	}

	mu.Unlock()
	writeJSON(w, etat) // envoie l'état mis à jour au front-end
}

// remet complètement à zéro la partie
func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { //Envoyer des données / modifier le serveur
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	nouveauPlateau()
	w.WriteHeader(http.StatusOK)
}

// colonnePleine vérifie si la colonne est pleine
func colonnePleine(col int) bool {
	return plateau[0][col] != 0 // si 1 ou 2 alors pleine
}

// place un jeton dans la colonne, à la première case disponible en partant du bas
func placerJeton(col int, joueur int) (int, error) { // verifie si la colonne est pleine
	for r := rows - 1; r >= 0; r-- { // parcours de -1 à 0
		if plateau[r][col] == 0 { // si case vide
			plateau[r][col] = joueur // le joueur place son jeton
			return r, nil            // confirmation
		}
	}
	return -1, fmt.Errorf("colonne pleine")
}

// isFull vérifie si tout le plateau est rempli
func isFull() bool {
	for r := 0; r < rows; r++ { // parcours de 0 tant que r < rows"T" on ajoute +1
		for c := 0; c < cols; c++ { // parcours de 0 tant que c < cols"T" on ajoute +1
			if plateau[r][c] == 0 {
				return false
			}
		}
	}
	return true
}

// vérifie si un joueur a aligné connectN jetons
func verifierVictoire() bool {
	// directions à explorer : droite, bas, diagonale bas-droite, diagonale bas-gauche
	dirs := [][2]int{ // xy axes
		{0, 1},  // droite
		{1, 0},  // bas
		{1, 1},  // diagonale bas-droite
		{1, -1}, // diagonale bas-gauche
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			val := plateau[r][c]
			if val == 0 {
				continue // case vide, on ignore
			}

			// on teste chaque direction
			for _, d := range dirs {
				cnt := 1       // compteur de jeton alignés
				nr := r + d[0] // nr nouvelles coordonnées en avançant dans la direction
				nc := c + d[1] // nc nouvelles coordonnées en avançant dans la direction

				// tant que les jetons sont alignés
				for nr >= 0 && nr < rows && nc >= 0 && nc < cols && plateau[nr][nc] == val {
					cnt++
					nr += d[0]
					nc += d[1]
				}

				if cnt >= connectN {
					return true // victoire détectée
				}
			}
		}
	}
	return false // aucune victoire
}
