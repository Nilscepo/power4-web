ignore

package p
// copie thématique : liaison HTTP / JSON / handlers

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
