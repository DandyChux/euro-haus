import { createFileRoute, Link } from '@tanstack/react-router';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Skeleton } from '~/components/ui/skeleton';
import { apiClient } from '~/lib/api';
import { CheckCircle, ArrowRight, Home } from 'lucide-react';
import { z } from 'zod';

const orderDetailSchema = z.object({
	id: z.string().startsWith("cs_"),
	amount: z.number().min(0),
	status: z.string().min(1),
	customer: z.object({
		email: z.string().email().or(z.string().default("")),
		name: z.string().min(1),
	}),
	items: z.array(
		z.object({
			id: z.string().startsWith("li_"),
			name: z.string().min(1),
			quantity: z.number().min(1),
			amount: z.number().min(0),
		})
	),
	created: z.number().min(0),
});

type OrderDetails = z.infer<typeof orderDetailSchema>;

export const Route = createFileRoute('/checkout/success')({
	component: CheckoutSuccessPage,
	loaderDeps: ({ search: { session_id } }) => ({
		sessionId: session_id,
	}),
	loader: async ({ deps: { sessionId } }) => {
		const response = await apiClient.get(`/checkout-session?session_id=${sessionId}`);
		return response.data;
	},
	pendingComponent: () => (
		<div className="container max-w-3xl mx-auto py-16 px-4">
			<Card>
				<CardHeader>
					<Skeleton className="h-8 w-64 mx-auto" />
				</CardHeader>
				<CardContent className="space-y-4">
					<Skeleton className="h-6 w-full" />
					<Skeleton className="h-6 w-3/4" />
					<Skeleton className="h-24 w-full" />
				</CardContent>
				<CardFooter>
					<Skeleton className="h-10 w-full" />
				</CardFooter>
			</Card>
		</div>
	),
	errorComponent: ({ error }) => (
		<div className="container max-w-3xl mx-auto py-16 px-4">
			<Card>
				<CardHeader>
					<CardTitle className="text-center text-2xl">Something went wrong: {error.name}</CardTitle>
				</CardHeader>
				<CardContent>
					<p className="text-center text-muted-foreground">
						{error.message}
					</p>
				</CardContent>
				<CardFooter className="flex justify-center gap-4">
					<Button asChild>
						<Link to="/">
							<Home className="mr-2 h-4 w-4" />
							Return Home
						</Link>
					</Button>
					<Button asChild variant="outline">
						<Link to="/catalog">
							Continue Shopping
						</Link>
					</Button>
				</CardFooter>
			</Card>
		</div>
	)
});

function CheckoutSuccessPage() {
	const loaderData = Route.useLoaderData()
	const order = orderDetailSchema.parse(loaderData)

	return (
		<div className="container max-w-3xl mx-auto py-16 px-4">
			<Card>
				<CardHeader>
					<div className="mx-auto w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mb-4">
						<CheckCircle className="h-10 w-10 text-primary" />
					</div>
					<CardTitle className="text-center text-3xl">Order Successful!</CardTitle>
				</CardHeader>
				<CardContent className="space-y-6">
					<div className="text-center">
						<p className="text-muted-foreground">
							Thank you for your purchase. We've received your order and are processing it now.
						</p>
						{order?.customer?.email && (
							<p className="font-medium mt-2">
								A confirmation email has been sent to {order.customer.email}
							</p>
						)}
					</div>

					{order && (
						<div className="bg-muted p-4 rounded-lg">
							<div className="flex mb-2 justify-between">
								<span className="font-medium">Order ID:</span>
								<span className="font-mono text-ellipsis">{order.id}</span>
							</div>
							<div className="flex justify-between">
								<span className="font-medium">Total Amount:</span>
								<span>${(order.amount / 100).toFixed(2)}</span>
							</div>
						</div>
					)}

					{order?.items && order.items.length > 0 && (
						<div>
							<h3 className="text-lg font-semibold mb-3">Order Summary</h3>
							<div className="divide-y">
								{order.items.map((item) => (
									<div key={item.id} className="py-3 flex justify-between">
										<div>
											<p className="font-medium">{item.name}</p>
											<p className="text-sm text-muted-foreground">Quantity: {item.quantity}</p>
										</div>
										<span>${(item.amount / 100).toFixed(2)}</span>
									</div>
								))}
							</div>
						</div>
					)}
				</CardContent>
				<CardFooter className="flex flex-col sm:flex-row gap-3 justify-center">
					<Button asChild className="sm:w-auto w-full">
						<Link to="/catalog">
							Continue Shopping
							<ArrowRight className="ml-2 h-4 w-4" />
						</Link>
					</Button>
					<Button asChild variant="outline" className="sm:w-auto w-full">
						<Link to="/">
							<Home className="mr-2 h-4 w-4" />
							Return Home
						</Link>
					</Button>
				</CardFooter>
			</Card>
		</div>
	);
}
