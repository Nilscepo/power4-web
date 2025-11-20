package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"
)

// Serveur simple Puissance 4
// Toute la logique est en Go, fonctions en français et commentaires en français.

var mu sync.Mutex

// plateau 6 lignes x 7 colonnes, 0 = vide, 1 = rouge, 2 = jaune
// plateau dynamique : rows x cols
var plateau [][]int
var rows int = 6
var cols int = 7
var connectN int = 4 // nombre à aligner pour gagner
var courant int = 1  // joueur courant: 1 ou 2
var vainqueur int = 0
var timers = map[int]int{1: 180, 2: 180} // secondes restantes pour chaque joueur
var egalite bool = false
var dernierRow int = -1
var dernierCol int = -1

func main() {
	nouveauPlateau()

	// goroutine pour le "chrono" : décrémente le temps du joueur courant
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			mu.Lock()
			if vainqueur == 0 && !egalite {
				if timers[courant] > 0 {
					timers[courant]--
					if timers[courant] <= 0 {
						// si le temps arrive à 0, fin de la partie en égalité
						egalite = true
					}
				}
			}
			mu.Unlock()
		}
	}()

	http.HandleFunc("/", handleIndex)
	// servir le dossier static sur /static/
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/state", handleState)
	http.HandleFunc("/play", handlePlay)
	http.HandleFunc("/set_mode", handleSetMode)
	http.HandleFunc("/reset", handleReset)
	http.HandleFunc("/game", handleGame)

	addr := ":8080"
	fmt.Println("Serveur démarré sur http://localhost" + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		// afficher l'erreur et tenter un port de secours
		fmt.Println("Erreur ListenAndServe:", err)
		fmt.Println("Tentative sur le port :8081...")
		if err2 := http.ListenAndServe(":8081", nil); err2 != nil {
			fmt.Println("Échec ListenAndServe sur :8081:", err2)
		}
	}
}

// nouveauPlateau réinitialise le jeu
func nouveauPlateau() {
	mu.Lock()
	defer mu.Unlock()
	// (re)créer le plateau selon rows x cols
	plateau = make([][]int, rows)
	for r := 0; r < rows; r++ {
		plateau[r] = make([]int, cols)
		for c := 0; c < cols; c++ {
			plateau[r][c] = 0
		}
	}
	courant = 1
	vainqueur = 0
	timers[1] = 180
	timers[2] = 180
	egalite = false
	dernierRow = -1
	dernierCol = -1
}

// handleIndex sert la page de menu (menu.html)
func handleIndex(w http.ResponseWriter, r *http.Request) {
	p, _ := filepath.Abs("./menu.html")
	http.ServeFile(w, r, p)
}

// handleGame sert la page du jeu (index.html)
func handleGame(w http.ResponseWriter, r *http.Request) {
	p, _ := filepath.Abs("./index.html")
	http.ServeFile(w, r, p)
}

// type pour réponse d'état
type Etat struct {
	Plateau    [][]int     `json:"plateau"`
	Courant    int         `json:"courant"`
	Vainqueur  int         `json:"vainqueur"`
	Timers     map[int]int `json:"timers"`
	DernierRow int         `json:"dernier_row"`
	DernierCol int         `json:"dernier_col"`
	Egalite    bool        `json:"egalite"`
}

func copyPlateau() [][]int {
	p := make([][]int, rows)
	for r := 0; r < rows; r++ {
		p[r] = make([]int, cols)
		for c := 0; c < cols; c++ {
			p[r][c] = plateau[r][c]
		}
	}
	return p
}

// handleState renvoie l'état JSON
func handleState(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	etat := Etat{Plateau: copyPlateau(), Courant: courant, Vainqueur: vainqueur, Timers: map[int]int{1: timers[1], 2: timers[2]}, DernierRow: dernierRow, DernierCol: dernierCol, Egalite: egalite}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(etat)
}

// structure pour la requête de jeu
type PlayReq struct {
	Col int `json:"col"`
}

// handlePlay place un jeton dans une colonne
func handlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req PlayReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	col := req.Col
	mu.Lock()
	defer mu.Unlock()
	var dernierR, dernierC = -1, -1
	if vainqueur != 0 {
		// jeu terminé
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "jeu termine"})
		return
	}
	if col < 0 || col >= cols {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "colonne invalide"})
		return
	}
	if colonnePleine(col) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "colonne pleine"})
		return
	}
	// placer jeton (renommer la variable pour éviter collision avec le paramètre *http.Request)
	ligne, err := placerJeton(col, courant)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "impossible de placer"})
		return
	}
	dernierR = ligne
	dernierC = col
	// mettre à jour dernier coup global
	dernierRow = dernierR
	dernierCol = dernierC
	// vérifier victoire
	if verifierVictoire() {
		vainqueur = courant
	} else {
		// si la grille est pleine -> égalité
		if isFull() {
			egalite = true
		} else {
			// changer joueur
			if courant == 1 {
				courant = 2
			} else {
				courant = 1
			}
		}
	}

	etat := Etat{Plateau: copyPlateau(), Courant: courant, Vainqueur: vainqueur, Timers: map[int]int{1: timers[1], 2: timers[2]}, DernierRow: dernierR, DernierCol: dernierC}
	etat.Egalite = egalite
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(etat)
}

// handleSetMode permet de configurer le mode de jeu (rows x cols ou noms prédéfinis)
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
	switch body.Mode {
	case "large":
		rows = 11
		cols = 9
		connectN = 4

	case "9x10":
		// mode alternatif demandé : 9 lignes x 10 colonnes
		rows = 9
		cols = 10
		connectN = 4
	case "normal":
		rows = 6
		cols = 7
		connectN = 4
	default:
		if body.Rows > 0 && body.Cols > 0 {
			rows = body.Rows
			cols = body.Cols
			connectN = 4
		}
	}
	// réinitialiser le plateau
	nouveauPlateau()
	w.WriteHeader(http.StatusOK)
}

// handleReset remet le jeu à zéro
func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	nouveauPlateau()
	w.WriteHeader(http.StatusOK)
}

// colonnePleine renvoie true si la colonne est pleine
func colonnePleine(col int) bool {
	// si la première ligne est non nulle, la colonne est pleine
	if rows == 0 || cols == 0 {
		return true
	}
	return plateau[0][col] != 0
}

// placerJeton place un jeton du joueur dans la colonne et renvoie la ligne
func placerJeton(col int, joueur int) (int, error) {
	for r := rows - 1; r >= 0; r-- {
		if plateau[r][col] == 0 {
			plateau[r][col] = joueur
			return r, nil
		}
	}
	return -1, fmt.Errorf("colonne pleine")
}

// isFull renvoie true si le plateau est plein
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

// verifierVictoire parcourt le plateau et detecte 4 à la suite
func verifierVictoire() bool {
	// directions : droite, bas, bas-droite, bas-gauche
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			val := plateau[r][c]
			if val == 0 {
				continue
			}
			for _, d := range dirs {
				cnt := 1
				nr := r + d[0]
				nc := c + d[1]
				for nr >= 0 && nr < rows && nc >= 0 && nc < cols && plateau[nr][nc] == val {
					cnt++
					nr += d[0]
					nc += d[1]
				}
				if cnt >= connectN {
					return true
				}
			}
		}
	}
	return false
}
