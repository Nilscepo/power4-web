package main

import (
	"encoding/json" // Pour encoder/décoder du JSON
	"fmt"           // Pour afficher du texte dans la console
	"net/http"      // Pour créer un serveur HTTP
	"path/filepath" // Pour gérer les chemins de fichiers
	"sync"          // Pour synchronisation via mutex (lock/unlock)
	"time"          // Pour gérer le temps, notamment les timers
)

// création de var. lock pour protéger les variables partagées
var mu sync.Mutex

// Variables globales représentant l'état du jeu
var (
	plateau    [][]int                       // le plateau, matrice de rows x cols contenant 0, 1, ou 2
	rows       = 6                           // nombre de lignes (std)
	cols       = 7                           // nombre de colonnes (std)
	connectN   = 4                           // nombre de jetons alignés nécessaires pour win
	courant    = 1                           // détermine quel joueur joue (1 ou 2)
	vainqueur  = 0                           // 0 = pas encore de vainqueur, 1 ou 2 sinon
	timers     = map[int]int{1: 180, 2: 180} // chronomètres pour chaque joueur (en sec.)
	egalite    = false                       // indique s'il y a égalité
	dernierRow = -1                          // dernière ligne où un jeton a été posé
	dernierCol = -1                          // dernière colonne où un jeton a été posé
)

func main() {
	nouveauPlateau() // initialise le plateau au démarrage

	// goroutine (permet la continuité du prog.) qui gère les timers (décrémentation chaque seconde)
	go func() {
		ticker := time.NewTicker(1 * time.Second) // un tick chaque seconde
		for range ticker.C {
			mu.Lock()                       // verrouille pour accès concurrent sécurisé
			if vainqueur == 0 && !egalite { // décrément uniquement si la partie continue
				if timers[courant] > 0 {
					timers[courant]-- // on enlève 1 seconde au joueur courant
					if timers[courant] <= 0 {
						// si le timer atteint 0, la partie est déclarée en égalité
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

// envoie la réponse JSON
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// utilitaire pour envoyer une erreur JSON
func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// handleState : renvoie l’état du jeu au client
func handleState(w http.ResponseWriter, r *http.Request) {
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

// PlayReq représente une requête où l’on indique la colonne à jouer
type PlayReq struct {
	Col int `json:"col"`
}

// handlePlay reçoit un coup du joueur
func handlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, http.StatusMethodNotAllowed, "méthode non autorisée")
		return
	}

	var req PlayReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}
	col := req.Col

	mu.Lock()

	// vérification si la partie est déjà finie
	if vainqueur != 0 {
		mu.Unlock()
		writeError(w, http.StatusConflict, "jeu termine")
		return
	}

	// vérification validité de la colonne
	if col < 0 || col >= cols {
		mu.Unlock()
		writeError(w, http.StatusBadRequest, "colonne invalide")
		return
	}

	// colonne pleine ?
	if colonnePleine(col) {
		mu.Unlock()
		writeError(w, http.StatusConflict, "colonne pleine")
		return
	}

	// placement du jeton dans la colonne
	ligne, err := placerJeton(col, courant)
	if err != nil {
		mu.Unlock()
		writeError(w, http.StatusInternalServerError, "impossible de placer")
		return
	}

	// mise à jour position du dernier jeton
	dernierRow = ligne
	dernierCol = col

	// vérification victoire, égalité, ou changement de joueur
	if verifierVictoire() {
		vainqueur = courant
	} else if isFull() {
		egalite = true
	} else {
		// changement de joueur
		if courant == 1 {
			courant = 2
		} else {
			courant = 1
		}
	}

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

// handleSetMode change la taille du plateau selon le mode choisi
func handleSetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Mode string `json:"mode"`
		Rows int    `json:"rows"`
		Cols int    `json:"cols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// sélection du mode
	switch body.Mode {
	case "large":
		rows = 11
		cols = 9
		connectN = 4

	case "9x10":
		rows = 9
		cols = 10
		connectN = 4

	case "normal":
		rows = 6
		cols = 7
		connectN = 4

	default:
		// mode personnalisé
		if body.Rows > 0 && body.Cols > 0 {
			rows = body.Rows
			cols = body.Cols
			connectN = 4
		}
	}

	// réinitialisation du plateau
	nouveauPlateau()

	w.WriteHeader(http.StatusOK)
}

// handleReset remet complètement à zéro la partie
func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	nouveauPlateau()
	w.WriteHeader(http.StatusOK)
}

// colonnePleine vérifie si la colonne est pleine
func colonnePleine(col int) bool {
	// une colonne est pleine si la première ligne est non vide
	if rows == 0 || cols == 0 {
		return true
	}
	return plateau[0][col] != 0
}

// placerJeton place un jeton dans la colonne, à la première case disponible en partant du bas
func placerJeton(col int, joueur int) (int, error) {
	for r := rows - 1; r >= 0; r-- {
		if plateau[r][col] == 0 {
			plateau[r][col] = joueur
			return r, nil
		}
	}
	return -1, fmt.Errorf("colonne pleine")
}

// isFull vérifie si tout le plateau est rempli
func isFull() bool {
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if plateau[r][c] == 0 {
				return false
			}
		}
	}
	return true
}

// verifierVictoire vérifie si un joueur a aligné connectN jetons
func verifierVictoire() bool {
	// directions à explorer : droite, bas, diagonale bas-droite, diagonale bas-gauche
	dirs := [][2]int{
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
				cnt := 1
				nr := r + d[0]
				nc := c + d[1]

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
