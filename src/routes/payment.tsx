import { Elements } from '@stripe/react-stripe-js'
import { createFileRoute } from '@tanstack/react-router'
import { loadStripe } from '@stripe/stripe-js'

export const Route = createFileRoute('/payment')({
	component: RouteComponent,
})

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);

function RouteComponent() {
	return (
		<Elements stripe={stripePromise}></Elements>
	)
}
