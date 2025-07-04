import React from 'react';
import { Button } from './ui/button';
import { apiClient } from '~/lib/api';

export function CheckoutButton({ priceId }: { priceId: string }) {
	const handleCheckout = async () => {
		try {
			// Call your backend to create a checkout session
			// const response = await fetch('/api/create-checkout-session', {
			// 	method: 'POST',
			// 	headers: { 'Content-Type': 'application/json' },
			// 	body: JSON.stringify({ priceId })
			// });
			const { data } = await apiClient.post('/create-checkout-session', { priceId });
			console.log(data)

			const { sessionId } = data;

			// Redirect to Stripe Checkout
			const stripe = window.Stripe?.(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);
			if (stripe) {
				await stripe.redirectToCheckout({ sessionId });
			}
		} catch (error) {
			console.error('Checkout error:', error);
		}
	};

	return (
		<Button onClick={handleCheckout}>Buy Now</Button>
	);
}
