package services

import (
	"context"

	"github.com/dandychux/euro-haus/internal/models"
)

func GetTicketByID(ctx context.Context, ticketID string) (*models.Ticket, error) {
	db := GetDB()
	var ticket models.Ticket
	err := db.WithContext(ctx).Where("token = ?", ticketID).First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}
