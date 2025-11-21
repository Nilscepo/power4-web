ignore

package p // copie thématique : UI / démarrage serveur

import (
	// Pour encoder/décoder du JSON
	"fmt"
	"net/http" // Pour gérer les chemins de fichiers
	"sync"     // Pour synchronisation via mutex (lock/unlock)
	"time"     // Pour gérer le temps, notamment les timers
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
