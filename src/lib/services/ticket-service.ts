import { apiClient } from '../api';

export interface Ticket {
	id: string;
	eventId: string;
	attendeeEmail: string;
	attendeeName: string;
	ticketType: string;
	ticketCode: string;
	checkedIn: boolean;
	checkedInAt?: string;
	purchasedAt: string;
}

export const ticketService = {
	async checkInTicket(ticketCode: string): Promise<Ticket> {
		try {
			const response = await apiClient.post<Ticket>(`/tickets/${ticketCode}/checkin`);
			return response.data;
		} catch (error) {
			console.error('Failed to check in ticket:', error);
			throw new Error('Failed to check in ticket');
		}
	},

	async getEventAttendees(eventId: string): Promise<Ticket[]> {
		try {
			const response = await apiClient.get<{ tickets: Ticket[] }>(`/events/${eventId}/tickets`);
			return response.data.tickets;
		} catch (error) {
			console.error('Failed to fetch attendees:', error);
			throw new Error('Failed to load attendees');
		}
	},

	generateTicketCode(): string {
		// Generate a unique ticket code
		const timestamp = Date.now().toString(36);
		const randomStr = Math.random().toString(36).substring(2, 9);
		return `TKT-${timestamp}-${randomStr}`.toUpperCase();
	},

	async validateTicket(ticketCode: string): Promise<Ticket | null> {
		try {
			const response = await apiClient.get<Ticket>(`/tickets/${ticketCode}`);
			return response.data;
		} catch (error) {
			return null;
		}
	}
};
