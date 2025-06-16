export interface Product {
	id: string;
	name: string;
	description: string | null;
	images: string[];
	price: number;
	currency: string;
	metadata: Record<string, string>;
}

export interface PaymentIntentResponse {
	clientSecret: string;
}
