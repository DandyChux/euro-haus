import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { Skeleton } from '~/components/ui/skeleton';
import { RefreshCw, AlertCircle, ArrowRight, Home, Car, User, Mail, Clock } from 'lucide-react';
import { z } from 'zod';
import { submissionService } from '~/lib/services/submission-service';
import { apiClient } from '~/lib/api';
import { useState } from 'react';
import { format } from 'date-fns';
import { Separator } from '~/components/ui/separator';
import { Badge } from '~/components/ui/badge';

const searchSchema = z.object({
	session: z.string().optional(),
	submission: z.string().optional(),
});

export const Route = createFileRoute('/checkout/recover')({
	component: CheckoutRecoverPage,
	validateSearch: searchSchema,
	loaderDeps: ({ search }) => ({
		sessionId: search.session,
		submissionId: search.submission,
	}),
	loader: async ({ deps: { sessionId, submissionId } }) => {
		// If we have a submission ID, try to load it
		if (submissionId) {
			try {
				const submission = await submissionService.getSubmission(submissionId);
				return {
					submission,
					sessionId: null,
					type: 'submission' as const
				};
			} catch (error) {
				console.error('Failed to load submission:', error);
				return {
					submission: null,
					sessionId: null,
					type: 'submission' as const,
					error: 'Could not find submission'
				};
			}
		}

		// If we have a session ID, we might want to recover a regular checkout
		if (sessionId) {
			return {
				submission: null,
				sessionId,
				type: 'session' as const
			};
		}

		return {
			submission: null,
			sessionId: null,
			type: null,
			error: 'No recovery information provided'
		};
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
			</Card>
		</div>
	),
});

function CheckoutRecoverPage() {
	const { submission, sessionId, type, error } = Route.useLoaderData();
	const navigate = useNavigate();
	const [isRecovering, setIsRecovering] = useState(false);
	const [recoveryError, setRecoveryError] = useState<string | null>(null);

	const handleSubmissionRecovery = async () => {
		if (!submission) return;

		setIsRecovering(true);
		setRecoveryError(null);

		try {
			// Get the event details to find the price
			const eventResponse = await apiClient.get(`/events/${submission.eventId}`);
			const event = eventResponse.data;

			// Find the appropriate ticket tier price
			let priceId = '';
			if (event.prices && event.prices.length > 0) {
				// Use the ticket tier from submission if available, otherwise use first price
				const ticketTier = submission.ticketTier || 'general';
				const price = event.prices.find((p: any) =>
					p.metadata?.tier === ticketTier
				) || event.prices[0];
				priceId = price.id;
			}

			if (!priceId) {
				throw new Error('Could not find pricing information for this event');
			}

			// Create a new checkout session for the submission
			const checkoutResponse = await apiClient.post('/create-participant-checkout', {
				submissionId: submission.id,
				priceId: priceId,
				quantity: submission.ticketQuantity || 1,
				eventName: event.name,
			});

			if (checkoutResponse.data.url) {
				// Redirect to Stripe checkout
				window.location.href = checkoutResponse.data.url;
			} else {
				throw new Error('Failed to create checkout session');
			}
		} catch (err: any) {
			console.error('Recovery failed:', err);
			setRecoveryError(
				err.response?.data?.message ||
				err.message ||
				'Failed to recover checkout session. Please try again.'
			);
			setIsRecovering(false);
		}
	};

	const handleSessionRecovery = async () => {
		// For regular cart recovery, redirect to cart page
		// The session ID might be expired, so we just send them to cart
		navigate({ to: '/cart' });
	};

	// Error state
	if (error) {
		return (
			<div className="container max-w-3xl mx-auto py-16 px-4">
				<Card>
					<CardHeader>
						<div className="mx-auto w-16 h-16 bg-destructive/10 rounded-full flex items-center justify-center mb-4">
							<AlertCircle className="h-10 w-10 text-destructive" />
						</div>
						<CardTitle className="text-center text-2xl">Recovery Failed</CardTitle>
					</CardHeader>
					<CardContent>
						<Alert variant="destructive">
							<AlertCircle className="h-4 w-4" />
							<AlertDescription>{error}</AlertDescription>
						</Alert>
						<p className="text-center text-muted-foreground mt-4">
							We couldn't find the information needed to recover your checkout session.
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
							<Link to="/events">
								View Events
							</Link>
						</Button>
					</CardFooter>
				</Card>
			</div>
		);
	}

	// Submission recovery
	if (type === 'submission' && submission) {
		return (
			<div className="container max-w-3xl mx-auto py-16 px-4">
				<Card>
					<CardHeader>
						<div className="mx-auto w-16 h-16 bg-orange-50 rounded-full flex items-center justify-center mb-4">
							<RefreshCw className="h-10 w-10 text-orange-500" />
						</div>
						<CardTitle className="text-center text-2xl">Complete Your Registration</CardTitle>
						<CardDescription className="text-center">
							Your payment session expired, but your submission is still saved
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-6">
						{recoveryError && (
							<Alert variant="destructive">
								<AlertCircle className="h-4 w-4" />
								<AlertDescription>{recoveryError}</AlertDescription>
							</Alert>
						)}

						{/* Submission Details */}
						<div className="bg-gray-50 rounded-lg p-4 space-y-3">
							<div className="flex items-center gap-2 text-sm font-medium">
								<Car className="h-4 w-4 text-muted-foreground" />
								Vehicle Details
							</div>
							<div className="pl-6 space-y-1">
								<p className="text-lg font-semibold">
									{submission.vehicleYear} {submission.vehicleMake} {submission.vehicleModel}
								</p>
								{submission.vehicleDescription && (
									<p className="text-sm text-muted-foreground">{submission.vehicleDescription}</p>
								)}
							</div>
						</div>

						<Separator />

						{/* Participant Information */}
						<div className="space-y-3">
							<h3 className="text-sm font-medium">Your Information</h3>
							<div className="grid gap-3 text-sm">
								<div className="flex items-center gap-2">
									<User className="h-4 w-4 text-muted-foreground" />
									<span>{submission.participantName}</span>
								</div>
								<div className="flex items-center gap-2">
									<Mail className="h-4 w-4 text-muted-foreground" />
									<span>{submission.participantEmail}</span>
								</div>
								<div className="flex items-center gap-2">
									<Clock className="h-4 w-4 text-muted-foreground" />
									<span>Originally submitted {format(new Date(submission.submittedAt), 'MMMM d, yyyy')}</span>
								</div>
							</div>
						</div>

						<Separator />

						{/* Status */}
						<div className="flex justify-center">
							<Badge variant="secondary" className="text-orange-600 bg-orange-50">
								<Clock className="h-3 w-3 mr-1" />
								Payment Session Expired - Ready to Restart
							</Badge>
						</div>
						{/* What happens next */}
						<div className="bg-blue-50 rounded-lg p-4">
							<h3 className="font-medium mb-2">What happens when you continue?</h3>
							<ol className="space-y-1 text-sm text-blue-900">
								<li>1. You'll be redirected to a new checkout session</li>
								<li>2. Your vehicle details are already saved</li>
								<li>3. Complete the payment process</li>
								<li>4. Wait for admin approval (usually 24-48 hours)</li>
								<li>5. Your payment will only be charged after approval</li>
							</ol>
						</div>
					</CardContent>
					<CardFooter className="flex flex-col gap-3">
						<Button
							onClick={handleSubmissionRecovery}
							disabled={isRecovering}
							className="w-full"
							size="lg"
						>
							{isRecovering ? (
								<>
									<RefreshCw className="mr-2 h-4 w-4 animate-spin" />
									Creating New Checkout Session...
								</>
							) : (
								<>
									<ArrowRight className="mr-2 h-4 w-4" />
									Continue to Payment
								</>
							)}
						</Button>
						<Button asChild variant="outline" className="w-full">
							<Link to="/events">
								Cancel and View Other Events
							</Link>
						</Button>
					</CardFooter>
				</Card>
			</div>
		);
	}

	// Regular session recovery (abandoned cart)
	if (type === 'session') {
		return (
			<div className="container max-w-3xl mx-auto py-16 px-4">
				<Card>
					<CardHeader>
						<div className="mx-auto w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mb-4">
							<RefreshCw className="h-10 w-10 text-primary" />
						</div>
						<CardTitle className="text-center text-2xl">Recover Your Cart</CardTitle>
						<CardDescription className="text-center">
							Your checkout session has expired, but your items may still be in your cart
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<p className="text-center text-muted-foreground">
							Click below to return to your cart and start a new checkout session.
						</p>

						<Alert>
							<AlertCircle className="h-4 w-4" />
							<AlertDescription>
								Note: If your cart is empty, you'll need to add your items again.
							</AlertDescription>
						</Alert>
					</CardContent>
					<CardFooter className="flex flex-col gap-3">
						<Button
							onClick={handleSessionRecovery}
							className="w-full"
							size="lg"
						>
							<ArrowRight className="mr-2 h-4 w-4" />
							Go to Cart
						</Button>
						<Button asChild variant="outline" className="w-full">
							<Link to="/catalog">
								Browse Products
							</Link>
						</Button>
					</CardFooter>
				</Card>
			</div>
		);
	}

	// Default fallback
	return (
		<div className="container max-w-3xl mx-auto py-16 px-4">
			<Card>
				<CardHeader>
					<CardTitle className="text-center text-2xl">Invalid Recovery Link</CardTitle>
				</CardHeader>
				<CardContent>
					<p className="text-center text-muted-foreground">
						This recovery link appears to be invalid or expired.
					</p>
				</CardContent>
				<CardFooter className="flex justify-center gap-4">
					<Button asChild>
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
