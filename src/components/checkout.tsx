import React, { useState } from 'react';
import { CardElement, useStripe, useElements } from '@stripe/react-stripe-js';
import type { PaymentIntentResponse } from '~/lib/interfaces/product';

interface CheckoutProps {
	productId: string;
	price: number;
	currency: string;
	onSuccess: () => void;
	onClose: () => void;
}

const Checkout: React.FC<CheckoutProps> = ({
	productId,
	price,
	currency,
	onSuccess,
	onClose
}) => {
	const stripe = useStripe();
	const elements = useElements();
	const [error, setError] = useState<string | null>(null);
	const [processing, setProcessing] = useState<boolean>(false);
	const [succeeded, setSucceeded] = useState<boolean>(false);

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setProcessing(true);
		setError(null);

		if (!stripe || !elements) {
			setError("Stripe not initialized");
			setProcessing(false);
			return;
		}

		try {
			// 1. Create Payment Intent
			const response = await fetch(`${import.meta.env.VITE_API_URL}/create-payment-intent`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ productId, quantity: 1 }),
			});

			if (!response.ok) {
				throw new Error(`Payment failed: ${response.statusText}`);
			}

			const { clientSecret }: PaymentIntentResponse = await response.json();

			// 2. Confirm Payment
			const result = await stripe.confirmCardPayment(clientSecret, {
				payment_method: {
					card: elements.getElement(CardElement)!,
					billing_details: {
						name: 'Customer Name', // Collect from form
					},
				},
			});

			if (result.error) {
				setError(result.error.message || "Payment failed");
				setProcessing(false);
			} else if (result.paymentIntent?.status === 'succeeded') {
				setSucceeded(true);
				onSuccess();
			}
		} catch (err) {
			setError(err instanceof Error ? err.message : "Payment processing error");
			setProcessing(false);
		}
	};

	return (
		<div className="checkout-modal">
			<div className="modal-content">
				<button className="close-button" onClick={onClose}>×</button>

				{succeeded ? (
					<div className="success-message">
						<h3>Payment Successful!</h3>
						<p>Your order has been processed.</p>
						<button onClick={onClose}>Continue Shopping</button>
					</div>
				) : (
					<form onSubmit={handleSubmit} className="payment-form">
						<h2>Complete Payment</h2>

						<div className="price-summary">
							<p>Total: ${(price / 100).toFixed(2)} {currency.toUpperCase()}</p>
						</div>

						<div className="card-section">
							<label>Card Details</label>
							<CardElement
								options={{
									style: {
										base: {
											fontSize: '16px',
											color: '#424770',
											'::placeholder': { color: '#aab7c4' },
										},
										invalid: { color: '#9e2146' },
									}
								}}
							/>
						</div>

						{error && <div className="error-message">{error}</div>}

						<div className="actions">
							<button
								type="submit"
								disabled={processing || !stripe}
								className="pay-button"
							>
								{processing ? 'Processing...' : `Pay $${(price / 100).toFixed(2)}`}
							</button>
							<button
								type="button"
								onClick={onClose}
								className="cancel-button"
							>
								Cancel
							</button>
						</div>
					</form>
				)}
			</div>
		</div>
	);
};

export default Checkout;
