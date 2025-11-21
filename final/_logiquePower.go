ignore

package p
// copie thématique : logique du jeu

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
