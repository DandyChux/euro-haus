package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
	"github.com/gorilla/mux"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var allowedRequirementTypes = map[string]bool{
	"text":     true,
	"textarea": true,
	"select":   true,
	"radio":    true,
	"boolean":  true,
	"number":   true,
}

type RequirementInput struct {
	ID       string   `json:"id"`
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options"`
	SortOrder int     `json:"sort_order"`
	Active   bool     `json:"active"`
}

func validateRequirementInput(input RequirementInput) error {
	input.Key = strings.TrimSpace(input.Key)
	input.Label = strings.TrimSpace(input.Label)
	input.Type = strings.TrimSpace(input.Type)

	if input.Key == "" {
		return fmt.Errorf("requirement key is required")
	}

	if input.Label == "" {
		return fmt.Errorf("requirement label is required")
	}

	if !allowedRequirementTypes[input.Type] {
		return fmt.Errorf("unsupported requirement type %q", input.Type)
	}

	if (input.Type == "select" || input.Type == "radio") &&
		len(input.Options) == 0 {
		return fmt.Errorf(
			"options are required for %s requirements",
			input.Type,
		)
	}

	return nil
}

func requirementFromInput(
	priceID string,
	input RequirementInput,
) models.PriceRequirement {
	options, _ := json.Marshal(input.Options)

	return models.PriceRequirement{
		ID:        input.ID,
		PriceID:   priceID,
		Key:       strings.TrimSpace(input.Key),
		Label:     strings.TrimSpace(input.Label),
		Type:      strings.TrimSpace(input.Type),
		Required:  input.Required,
		Options:   datatypes.JSON(options),
		SortOrder: input.SortOrder,
		Active:    input.Active,
	}
}

func loadPriceRequirements(
	tx *gorm.DB,
	priceID string,
) ([]models.PriceRequirement, error) {
	var requirements []models.PriceRequirement

	err := tx.
		Where("price_id = ? AND active = TRUE", priceID).
		Order("sort_order ASC, id ASC").
		Find(&requirements).
		Error

	return requirements, err
}

func loadPriceRequirementsForPrice(
	priceID string,
) ([]models.PriceRequirement, error) {
	return loadPriceRequirements(
		services.GetDB(),
		priceID,
	)
}

func GetPriceRequirements(w http.ResponseWriter, r *http.Request) {
	priceID := mux.Vars(r)["priceId"]

	var requirements []models.PriceRequirement

	err := services.GetDB().
		WithContext(r.Context()).
		Where("price_id = ? AND active = TRUE", priceID).
		Order("sort_order ASC, id ASC").
		Find(&requirements).
		Error

	if err != nil {
		http.Error(
			w,
			"Unable to load price requirements",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"requirements": requirements,
	})
}

func ReplacePriceRequirements(w http.ResponseWriter, r *http.Request) {
	priceID := mux.Vars(r)["priceId"]

	var request struct {
		Requirements []RequirementInput `json:"requirements"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for _, input := range request.Requirements {
		if err := validateRequirementInput(input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	db := services.GetDB().WithContext(r.Context())

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&models.PriceRequirement{}).
			Where("price_id = ?", priceID).
			Update("active", false).
			Error; err != nil {
			return err
		}

		for _, input := range request.Requirements {
			requirement := requirementFromInput(priceID, input)

			if requirement.ID == "" {
				if err := tx.Create(&requirement).Error; err != nil {
					return err
				}
				continue
			}

			var existing models.PriceRequirement
			if err := tx.
				Where("id = ? AND price_id = ?", requirement.ID, priceID).
				First(&existing).
				Error; err != nil {
				return err
			}

			if err := tx.
				Model(&existing).
				Updates(map[string]any{
					"key":        requirement.Key,
					"label":      requirement.Label,
					"type":       requirement.Type,
					"required":   requirement.Required,
					"options":    requirement.Options,
					"sort_order": requirement.SortOrder,
					"active":     true,
				}).
				Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		http.Error(
			w,
			"Unable to save price requirements",
			http.StatusInternalServerError,
		)
		return
	}

	requirements, err := loadPriceRequirements(db, priceID)
	if err != nil {
		http.Error(
			w,
			"Unable to reload price requirements",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"requirements": requirements,
	})
}
