export interface VehicleSubmission {
	id: string;
	eventId: string;
	eventSlug: string;
	participantName: string;
	participantEmail: string;
	participantPhone?: string;
	vehicleYear: string;
	vehicleMake: string;
	vehicleModel: string;
	vehicleDescription?: string;
	vehicleModifications?: string;
	images: string[]; // URLs to images in Spaces
	status: 'pending' | 'approved' | 'denied';
	submittedAt: string;
	reviewedAt?: string;
	reviewedBy?: string;
	reviewNotes?: string;
	checkoutSessionId?: string;
	paymentIntentId?: string;
	ticketId?: string;
	ticketTier?: string;
	ticketPrice?: number;
	ticketQuantity?: number;
}

export interface SubmissionCreateRequest {
	eventId: string;
	eventSlug: string;
	participantName: string;
	participantEmail: string;
	participantPhone?: string;
	vehicleYear: string;
	vehicleMake: string;
	vehicleModel: string;
	vehicleDescription?: string;
	vehicleModifications?: string;
	ticketTier?: string;
	ticketPrice?: number;
	ticketQuantity?: number;
}
