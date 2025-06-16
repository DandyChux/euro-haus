export interface Event {
	title: string;
	slug: string;
	date: string;
	description: string;
	location: string;
	price: number;
	image: string;
	capacity?: number;
	availableSpots?: number;
	organizer?: string;
	agenda?: {
		time: string;
		activity: string;
	}[];
	includes?: string[];
	tags?: string[];
	status?: 'upcoming' | 'ongoing' | 'completed' | 'cancelled' | 'soldout';
	createdAt?: string;
	updatedAt?: string;
}
