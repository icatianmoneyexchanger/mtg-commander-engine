package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// DeckCard represents an individual card entry in a Commander deck
type DeckCard struct {
	Name       string   `json:"name"`
	Quantity   int      `json:"quantity"`
	CMC        float64  `json:"cmc"`
	TypeLine   string   `json:"type_line"`
	ManaCost   string   `json:"mana_cost"`
	Categories []string `json:"categories"` // Ramp, Draw, Removal, BoardWipe, Synergy, Land
}

// CommanderDeck represents a full 100-card EDH deck
type CommanderDeck struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Commander   DeckCard   `json:"commander"`
	Cards       []DeckCard `json:"cards"`
	CreatedDate time.Time  `json:"created_date"`
}

// DeckAuditReport provides feedback based on the Master Commander Deckbuilding Framework
type DeckAuditReport struct {
	TotalCards       int      `json:"total_cards"`
	LandCount        int      `json:"land_count"`
	RampCount        int      `json:"ramp_count"`
	CardDrawCount    int      `json:"card_draw_count"`
	RemovalCount     int      `json:"removal_count"`
	BoardWipeCount   int      `json:"board_wipe_count"`
	AverageCMC       float64  `json:"average_cmc"`
	HealthScore      int      `json:"health_score"` // 0 to 100
	Recommendations  []string `json:"recommendations"`
	PremiumUnlocked  bool     `json:"premium_unlocked"`
}

type EngineStore struct {
	mu    sync.RWMutex
	Decks map[string]CommanderDeck
}

var store = &EngineStore{
	Decks: make(map[string]CommanderDeck),
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8095"
	}

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/v1/deck/audit", handleAuditDeck)
	http.HandleFunc("/api/v1/guide/modules", handleGuideModules)

	log.Printf("MTG Commander Deckbuilding & Monetization Engine active on port %s", port)
	log.Printf("Listening for Cloud6 MTG-ARM queries on http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Engine failed to start: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":   "Cloud6 MTG Commander Engine",
		"status":    "active",
		"timestamp": time.Now().UTC(),
	})
}

// Audit algorithm enforcing core EDH proportions (36-38 Lands, 10+ Ramp, 10+ Draw, 10+ Removal)
func handleAuditDeck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var deck CommanderDeck
	if err := json.NewDecoder(r.Body).Decode(&deck); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report := auditCommanderDeck(deck)
	json.NewEncoder(w).Encode(report)
}

func auditCommanderDeck(deck CommanderDeck) DeckAuditReport {
	report := DeckAuditReport{
		Recommendations: make([]string, 0),
		HealthScore:     100,
	}

	totalCards := 1 // Commander
	totalCMC := 0.0
	nonLandCards := 0

	for _, c := range deck.Cards {
		totalCards += c.Quantity

		isLand := strings.Contains(strings.ToLower(c.TypeLine), "land")
		if isLand {
			report.LandCount += c.Quantity
		} else {
			nonLandCards += c.Quantity
			totalCMC += c.CMC * float64(c.Quantity)
		}

		for _, cat := range c.Categories {
			catLower := strings.ToLower(cat)
			switch catLower {
			case "ramp":
				report.RampCount += c.Quantity
			case "draw", "card_draw":
				report.CardDrawCount += c.Quantity
			case "removal", "interaction":
				report.RemovalCount += c.Quantity
			case "boardwipe", "board_wipe":
				report.BoardWipeCount += c.Quantity
			}
		}
	}

	report.TotalCards = totalCards
	if nonLandCards > 0 {
		report.AverageCMC = totalCMC / float64(nonLandCards)
	}

	// --- Evaluator Rules ---
	if report.LandCount < 36 {
		penalty := (36 - report.LandCount) * 4
		report.HealthScore -= penalty
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("⚠️ Land count is low (%d/36 recommended minimum). Add more lands to avoid mana screw.", report.LandCount))
	}

	if report.RampCount < 10 {
		penalty := (10 - report.RampCount) * 3
		report.HealthScore -= penalty
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("⚠️ Ramp count is low (%d/10 recommended). Consider adding Sol Ring, Arcane Signet, or mana dorks.", report.RampCount))
	}

	if report.CardDrawCount < 10 {
		penalty := (10 - report.CardDrawCount) * 3
		report.HealthScore -= penalty
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("⚠️ Card draw engines are low (%d/10 recommended). Add repeatable draw sources like Rhystic Study or Mystic Remora.", report.CardDrawCount))
	}

	if report.RemovalCount < 8 {
		penalty := (8 - report.RemovalCount) * 4
		report.HealthScore -= penalty
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("⚠️ Instant-speed interaction is low (%d/8 recommended). Add spot removal like Counterspell or Cyclonic Rift.", report.RemovalCount))
	}

	if report.AverageCMC > 3.5 {
		report.HealthScore -= 10
		report.Recommendations = append(report.Recommendations, fmt.Sprintf("⚠️ Average mana curve is high (%.2f CMC). Lower your average CMC below 3.3 for faster tempo.", report.AverageCMC))
	}

	if report.HealthScore < 0 {
		report.HealthScore = 0
	}

	return report
}

func handleGuideModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	modules := []map[string]interface{}{
		{
			"id":          "mod_01",
			"title":       "Pillar 1: Building a Resilient Mana Base",
			"description": "Master land-to-spells ratios, color fixing, and curve optimization for Mono & Multi-color commanders.",
			"price_usd":   "4.99",
			"is_free":     true,
		},
		{
			"id":          "mod_02",
			"title":       "Pillar 2: The 8x8 Rule & Functional Categorization",
			"description": "Break your 99 cards into 8 functional categories of 8 cards to guarantee high consistency every game.",
			"price_usd":   "9.99",
			"is_free":     false,
		},
		{
			"id":          "mod_03",
			"title":       "Pillar 3: Infinite Combos & Win-Condition Layering",
			"description": "How to execute clean 2-card and 3-card win conditions with protection in Mono-Blue and competitive pod play.",
			"price_usd":   "14.99",
			"is_free":     false,
		},
	}
	json.NewEncoder(w).Encode(modules)
}
