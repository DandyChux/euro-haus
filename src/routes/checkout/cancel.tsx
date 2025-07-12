import { createFileRoute, Link } from '@tanstack/react-router';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { ShoppingCart, XCircle, ArrowLeft } from 'lucide-react';

export const Route = createFileRoute('/checkout/cancel')({
	component: CheckoutCancelPage,
});

function CheckoutCancelPage() {
	return (
		<div className="container max-w-3xl mx-auto py-16 px-4">
			<Card>
				<CardHeader>
					<div className="mx-auto w-16 h-16 bg-muted rounded-full flex items-center justify-center mb-4">
						<XCircle className="h-10 w-10 text-muted-foreground" />
					</div>
					<CardTitle className="text-center text-3xl">Checkout Canceled</CardTitle>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="text-center">
						<p className="text-muted-foreground">
							Your checkout process was canceled and no payment has been made.
						</p>
						<p className="mt-2">
							Your cart items are still saved, and you can continue your purchase whenever you're ready.
						</p>
					</div>

					<div className="bg-muted p-4 rounded-lg">
						<div className="flex items-center gap-3">
							<ShoppingCart className="h-5 w-5 text-muted-foreground" />
							<p>Need help with your order? Contact our support team for assistance.</p>
						</div>
					</div>
				</CardContent>
				<CardFooter className="flex flex-col sm:flex-row gap-3 justify-center">
					<Button asChild className="sm:w-auto w-full">
						<Link to="/cart">
							<ShoppingCart className="mr-2 h-4 w-4" />
							Return to Cart
						</Link>
					</Button>
					<Button asChild variant="outline" className="sm:w-auto w-full">
						<Link to="/catalog">
							<ArrowLeft className="mr-2 h-4 w-4" />
							Continue Shopping
						</Link>
					</Button>
				</CardFooter>
			</Card>
		</div>
	);
}
