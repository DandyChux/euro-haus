import React from 'react';
import { Button } from './ui/button';

export function CheckoutButton({ priceId }: { priceId: string }) {
	const handleCheckout = async () => {
		try {
			// Call your backend to create a checkout session
			const response = await fetch('/api/create-checkout-session', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ priceId })
			});

			const { sessionId } = await response.json();

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
