import { apiClient } from '../api';

export interface Ticket {
	id: string;
	eventId: string;
	eventName?: string;
	attendeeEmail: string;
	attendeeName: string;
	ticketType: string;
	ticketCode: string;
	checkedIn: boolean;
	checkedInAt?: string;
	purchasedAt: string;
}

export interface TicketValidationResponse {
	valid: boolean;
	message?: string;
	customerName?: string;
	customerEmail?: string;
	eventName?: string;
	eventId?: string;
	productId?: string;
	quantity?: number;
	checkedIn?: boolean;
	checkedInAt?: string;
	ticketType?: string;
	attendeeName?: string;
	attendeeEmail?: string;
	ticketCode?: string;
}

export const ticketService = {
	async checkInTicket(ticketCode: string): Promise<Ticket> {
		try {
			const response = await apiClient.post<TicketValidationResponse>(`/admin/events/ticket/check-in`, {
				token: ticketCode
			});

			// Transform the response to match Ticket interface
			return {
				id: ticketCode,
				eventId: response.data.productId || response.data.eventId || '',
				eventName: response.data.eventName,
				attendeeEmail: response.data.customerEmail || response.data.attendeeEmail || '',
				attendeeName: response.data.customerName || response.data.attendeeName || '',
				ticketType: response.data.ticketType || 'General',
				ticketCode: ticketCode,
				checkedIn: true,
				checkedInAt: response.data.checkedInAt || new Date().toISOString(),
				purchasedAt: '',
			};
		} catch (error) {
			console.error('Failed to check in ticket:', error);
			throw new Error('Failed to check in ticket');
		}
	},

	async getEventAttendees(eventId: string): Promise<Ticket[]> {
		try {
			const response = await apiClient.get<{ tickets: Ticket[] }>(`/events/${eventId}/tickets`);
			return response.data.tickets || [];
		} catch (error) {
			console.error('Failed to fetch attendees:', error);
			return [];
		}
	},

	generateTicketCode(): string {
		// Generate a unique ticket code
		const timestamp = Date.now().toString(36);
		const randomStr = Math.random().toString(36).substring(2, 9);
		return `TKT-${timestamp}-${randomStr}`.toUpperCase();
	},

	async validateTicket(ticketCode: string): Promise<TicketValidationResponse | null> {
		try {
			const response = await apiClient.post<TicketValidationResponse>(`/events/ticket/validate`, {
				token: ticketCode
			});

			if (!response.data.valid) {
				return null;
			}

			// Map the response to include all necessary fields
			return {
				...response.data,
				eventId: response.data.productId || response.data.eventId,
				attendeeName: response.data.customerName || response.data.attendeeName || '',
				attendeeEmail: response.data.customerEmail || response.data.attendeeEmail || '',
				ticketCode: ticketCode,
				ticketType: response.data.ticketType || 'General'
			};
		} catch (error) {
			console.error('Ticket validation error:', error);
			return null;
		}
	}
};
