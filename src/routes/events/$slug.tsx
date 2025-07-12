import { createFileRoute } from '@tanstack/react-router';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';
import { Link } from '@tanstack/react-router';
import { Calendar, MapPin, Users, Clock, ChevronLeft, Share2, Heart, Plus, Minus, Car } from 'lucide-react';
import { Badge } from '~/components/ui/badge';
import { Separator } from '~/components/ui/separator';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Input } from '~/components/ui/input';
import { stripeService, TieredPrice, EventWithTiers } from '~/lib/services/stripe-service';
import { useCart } from '~/lib/contexts/cart-context';
import { useState } from 'react';
import { toast } from 'sonner';
import { InfiniteMovingCards } from '~/components/ui/infinite-moving-cards';
import { apiClient } from '~/lib/api';
import { TieredPricing } from '~/components/tiered-pricing';
import { Dialog, DialogContent } from '~/components/ui/dialog';
import { VehicleSubmissionForm } from '~/components/vehicle-submission-form';
import { Checkbox } from '~/components/ui/checkbox';
import { Label } from '~/components/ui/label';

export const Route = createFileRoute('/events/$slug')({
	loader: async ({ params }) => {
		// Try to fetch event with tiers first
		const eventWithTiers = await stripeService.getEventWithPriceTiers(params.slug);
		if (eventWithTiers) {
			return { event: eventWithTiers, hasTiers: true };
		}

		// Fallback to regular event
		const event = await stripeService.getEventBySlug(params.slug);
		if (!event) {
			throw new Error('Event not found');
		}
		return { event, hasTiers: false };
	},
	component: EventDetailPage,
});

function EventDetailPage() {
	const { event, hasTiers } = Route.useLoaderData();
	const { addItem } = useCart();
	const [quantity, setQuantity] = useState(1);
	const [selectedTier, setSelectedTier] = useState<TieredPrice | null>(null);
	const [isParticipant, setIsParticipant] = useState(false);
	const [showSubmissionForm, setShowSubmissionForm] = useState(false);
	const [submissionId, setSubmissionId] = useState<string | null>(null);

	const handleSelectTier = async (tier: TieredPrice, tierQuantity: number) => {
		// If participant checkbox is selected, show submission form instead
		if (isParticipant) {
			setSelectedTier(tier);
			setQuantity(tierQuantity);
			setShowSubmissionForm(true);
			return;
		}

		try {
			const response = await apiClient.post('/create-checkout-session', {
				line_items: [
					{
						price: tier.priceId,
						quantity: tierQuantity,
					}
				],
				success_url: `${window.location.origin}/checkout/success`,
				cancel_url: `${window.location.origin}/checkout/cancel`,
				metadata: {
					event_id: event?.id,
					event_name: event?.title,
					tier_name: tier.name,
				},
			});

			const stripe = window.Stripe?.(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);
			if (stripe && response.data.sessionId) {
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			}
		} catch (error) {
			console.error('Checkout error:', error);
			toast.error('Failed to start checkout');
		}
	};

	const handleSingleCheckout = async () => {
		// If participant checkbox is selected, create a special checkout session
		if (isParticipant) {
			try {
				// First, show the submission form
				setShowSubmissionForm(true);
				return;
			} catch (error) {
				console.error('Error:', error);
				toast.error('Failed to start submission process');
			}
			return;
		}

		// Regular checkout flow continues...
		try {
			const response = await apiClient.post('/create-checkout-session', {
				priceId: event?.priceId,
				quantity,
			});

			const stripe = window.Stripe?.(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);
			if (stripe && response.data.sessionId) {
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			}
		} catch (error) {
			console.error('Checkout error:', error);
			toast.error('Failed to start checkout');
		}
	};

	const handleSubmissionSuccess = async (newSubmissionId: string) => {
		setSubmissionId(newSubmissionId);
		setShowSubmissionForm(false);

		// Create checkout session with manual capture for the submission
		try {
			const priceId = selectedTier?.priceId || event?.priceId;
			const response = await apiClient.post('/create-participant-checkout', {
				submissionId: newSubmissionId,
				priceId: priceId,
				eventName: event?.title,
				quantity: quantity,
			});

			const stripe = window.Stripe?.(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);
			if (stripe && response.data.sessionId) {
				// Redirect to Stripe checkout
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			}
		} catch (error) {
			console.error('Checkout error:', error);
			toast.error('Failed to create checkout session');
		}
	};


	const handleTierSelection = (tier: TieredPrice) => {
		setSelectedTier(tier);
		setQuantity(1); // Reset quantity when tier changes
	};

	const handleAddToCart = () => {
		if (event.status === 'soldout') {
			toast.error('This event is sold out');
			return;
		}

		// If participant checkbox is selected, show submission form instead
		if (isParticipant) {
			setShowSubmissionForm(true);
			return;
		}

		// If has tiers and no tier selected
		if (hasTiers && (event as EventWithTiers).priceTiers && !selectedTier) {
			toast.error('Please select a ticket type');
			return;
		}

		const ticketPrice = selectedTier ? selectedTier.amount : event.price;
		const ticketType = selectedTier ? selectedTier.name : 'General Admission';
		const maxAvailable = selectedTier?.maxQuantity ?
			Math.min(selectedTier.maxQuantity, event.availableSpots || 10) :
			(event.availableSpots || event.maxQuantity || 10);

		if (quantity > maxAvailable) {
			toast.error(`Only ${maxAvailable} tickets remaining for this type`);
			return;
		}

		addItem({
			id: event.id,
			priceId: selectedTier?.priceId || event.priceId,
			title: `${event.title} - ${ticketType}`,
			description: `Event on ${new Date(event.date).toLocaleDateString()}`,
			price: ticketPrice,
			quantity,
			imageUrl: event.imageUrl,
			maxQuantity: maxAvailable,
			type: 'event',
			eventDate: event.date,
		});

		toast.success(`Added ${quantity} ${ticketType} ticket${quantity > 1 ? 's' : ''} to cart`);
	};

	const handleShare = async () => {
		if (navigator.share) {
			try {
				await navigator.share({
					title: event.title,
					text: event.description,
					url: window.location.href,
				});
			} catch (error) {
				console.log('Error sharing:', error);
			}
		} else {
			navigator.clipboard.writeText(window.location.href);
			toast.success('Link copied to clipboard');
		}
	};

	if (!event) {
		return (
			<div className="min-h-screen flex items-center justify-center">
				<div className="text-center">
					<h2 className="text-2xl font-bold mb-4">Event not found</h2>
					<Button asChild>
						<Link to="/events">
							<ChevronLeft className="mr-2 h-4 w-4" />
							Back to Events
						</Link>
					</Button>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen">
			{/* Hero Image */}
			<div className="relative h-[400px] overflow-hidden">
				<Image
					src={event.imageUrl}
					alt={event.title}
					className="w-full h-full object-cover"
				/>
				<div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent" />
				<div className="absolute bottom-0 left-0 right-0 p-6 text-white">
					<div className="max-w-7xl mx-auto">
						<div className="flex items-center gap-4 mb-4">
							{event.tags?.map((tag) => (
								<Badge key={tag} variant="secondary" className="bg-white/20 text-white border-white/30">
									{tag}
								</Badge>
							))}
						</div>
						<h1 className="text-4xl font-bold mb-2">{event.title}</h1>
						<p className="text-lg opacity-90">{event.description}</p>
					</div>
				</div>
			</div>

			{/* Main Content */}
			<div className="max-w-7xl mx-auto px-6 py-8">
				<div className="grid lg:grid-cols-3 gap-8">
					{/* Left Column - Main Info */}
					<div className="lg:col-span-2 space-y-8">
						{/* Event Details */}
						<Card>
							<CardHeader>
								<CardTitle>Event Details</CardTitle>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="flex items-center gap-3">
									<Calendar className="h-5 w-5 text-muted-foreground" />
									<div>
										<p className="font-medium">Date</p>
										<p className="text-sm text-muted-foreground">
											{new Date(event.date).toLocaleDateString('en-US', {
												weekday: 'long',
												year: 'numeric',
												month: 'long',
												day: 'numeric'
											})}
										</p>
									</div>
								</div>
								<Separator />
								<div className="flex items-center gap-3">
									<MapPin className="h-5 w-5 text-muted-foreground" />
									<div>
										<p className="font-medium">Location</p>
										<p className="text-sm text-muted-foreground">{event.location}</p>
									</div>
								</div>
								<Separator />
								<div className="flex items-center gap-3">
									<Users className="h-5 w-5 text-muted-foreground" />
									<div>
										<p className="font-medium">Capacity</p>
										<p className="text-sm text-muted-foreground">
											{event.availableSpots} of {event.capacity} spots available
										</p>
									</div>
								</div>
								{event.organizer && (
									<>
										<Separator />
										<div>
											<p className="font-medium">Organized by</p>
											<p className="text-sm text-muted-foreground">{event.organizer}</p>
										</div>
									</>
								)}
							</CardContent>
						</Card>

						{/* Agenda */}
						{event.agenda && event.agenda.length > 0 && (
							<Card>
								<CardHeader>
									<CardTitle>Event Schedule</CardTitle>
								</CardHeader>
								<CardContent>
									<div className="space-y-4">
										{event.agenda.map((item, index) => (
											<div key={index} className="flex gap-4">
												<div className="flex items-center gap-2 min-w-[100px]">
													<Clock className="h-4 w-4 text-muted-foreground" />
													<span className="text-sm font-medium">{item.time}</span>
												</div>
												<div className="flex-1">
													<p className="text-sm">{item.activity}</p>
												</div>
											</div>
										))}
									</div>
								</CardContent>
							</Card>
						)}

						{/* What's Included */}
						{event.includes && event.includes.length > 0 && (
							<Card>
								<CardHeader>
									<CardTitle>What's Included</CardTitle>
								</CardHeader>
								<CardContent>
									<ul className="space-y-2">
										{event.includes.map((item, index) => (
											<li key={index} className="flex items-center gap-2">
												<span className="text-primary mt-1">•</span>
												<span className="text-sm">{item}</span>
											</li>
										))}
									</ul>
								</CardContent>
							</Card>
						)}
					</div>

					{/* Right Column - Booking */}
					<div className="space-y-4">
						{/* Participant Option Card */}
						<Card className="border-primary/20 bg-primary/5">
							<CardContent className="pt-6">
								<div className="flex items-start space-x-3">
									<Checkbox
										id="participant"
										checked={isParticipant}
										onCheckedChange={(checked) => setIsParticipant(checked as boolean)}
									/>
									<div className="space-y-1">
										<Label htmlFor="participant" className="cursor-pointer">
											<div className="flex items-center gap-2">
												<Car className="h-4 w-4" />
												<span className="font-medium">I want to participate as a presenter</span>
											</div>
										</Label>
										<p className="text-sm text-muted-foreground">
											Submit your vehicle for review. Payment will be processed after approval.
										</p>
									</div>
								</div>
							</CardContent>
						</Card>

						{/* Tiered Pricing */}
						{hasTiers && (event as EventWithTiers).priceTiers && (event as EventWithTiers).priceTiers.length > 0 ? (
							<div className="space-y-4">
								<Card>
									<CardHeader>
										<CardTitle>Select Tickets</CardTitle>
									</CardHeader>
									<CardContent>
										<TieredPricing
											tiers={(event as EventWithTiers).priceTiers}
											onSelectTier={(tier, qty) => {
												handleTierSelection(tier);
												handleSelectTier(tier, qty);
											}}
										/>
									</CardContent>
								</Card>

								{/* Add to Cart for Tiered Events */}
								{selectedTier && (
									<Card>
										<CardHeader>
											<CardTitle>Add to Cart</CardTitle>
										</CardHeader>
										<CardContent className="space-y-4">
											<div className="text-lg">
												<span className="font-semibold">{selectedTier.name}</span>
												<span className="text-muted-foreground"> - ${selectedTier.amount}</span>
											</div>

											{/* Quantity Selector */}
											<div className="space-y-2">
												<label className="text-sm font-medium">Number of Tickets</label>
												<div className="flex items-center gap-2">
													<Button
														variant="outline"
														size="icon"
														className="h-8 w-8"
														onClick={() => setQuantity(Math.max(1, quantity - 1))}
														disabled={quantity <= 1}
													>
														<Minus className="h-3 w-3" />
													</Button>
													<Input
														type="number"
														value={quantity}
														onChange={(e) => setQuantity(Math.max(1, Math.min(selectedTier.maxQuantity || 10, parseInt(e.target.value) || 1)))}
														className="w-16 h-8 text-center"
														min="1"
														max={selectedTier.maxQuantity || 10}
													/>
													<Button
														variant="outline"
														size="icon"
														className="h-8 w-8"
														onClick={() => setQuantity(Math.min(selectedTier.maxQuantity || 10, quantity + 1))}
														disabled={quantity >= (selectedTier.maxQuantity || 10)}
													>
														<Plus className="h-3 w-3" />
													</Button>
												</div>
											</div>

											<Button
												className="w-full"
												size="lg"
												onClick={handleAddToCart}
											>
												{isParticipant ? 'Submit Vehicle for Review' : `Add to Cart - $${(selectedTier.amount * quantity).toFixed(2)}`}
											</Button>
										</CardContent>
									</Card>
								)}
							</div>
						) : (
							/* Single Price Booking */
							<Card className="sticky top-24">
								<CardHeader>
									<CardTitle>Book Your Spot</CardTitle>
								</CardHeader>
								<CardContent className="space-y-4">
									<div className="text-3xl font-bold">
										${event.price.toFixed(2)}
										<span className="text-sm font-normal text-muted-foreground"> per ticket</span>
									</div>

									{/* Quantity Selector */}
									<div className="space-y-2">
										<label className="text-sm font-medium">Number of Tickets</label>
										<div className="flex items-center gap-2">
											<Button
												variant="outline"
												size="icon"
												className="h-8 w-8"
												onClick={() => setQuantity(Math.max(1, quantity - 1))}
												disabled={quantity <= 1}
											>
												<Minus className="h-3 w-3" />
											</Button>
											<Input
												type="number"
												value={quantity}
												onChange={(e) => setQuantity(Math.max(1, Math.min(event.availableSpots || 10, parseInt(e.target.value) || 1)))}
												className="w-16 h-8 text-center"
												min="1"
												max={event.availableSpots || 10}
											/>
											<Button
												variant="outline"
												size="icon"
												className="h-8 w-8"
												onClick={() => setQuantity(Math.min(event.availableSpots || 10, quantity + 1))}
												disabled={quantity >= (event.availableSpots || 10)}
											>
												<Plus className="h-3 w-3" />
											</Button>
										</div>
									</div>

									<Separator />

									<div className="space-y-2 text-sm">
										<p className="flex justify-between">
											<span>Ticket Price</span>
											<span className="font-medium">${event.price.toFixed(2)} × {quantity}</span>
										</p>
										<p className="flex justify-between">
											<span>Processing Fee</span>
											<span className="font-medium">$0.00</span>
										</p>
										<Separator />
										<p className="flex justify-between text-base font-bold">
											<span>Total</span>
											<span>${(event.price * quantity).toFixed(2)}</span>
										</p>
									</div>

									<div className="space-y-2">
										<Button
											className="w-full"
											size="lg"
											onClick={handleSingleCheckout}
											disabled={event.status === 'soldout' || event.availableSpots === 0}
										>
											{isParticipant
												? 'Submit Vehicle for Review'
												: (event.status === 'soldout' ? 'Sold Out' : 'Book Now')
											}
										</Button>
										{!isParticipant && (
											<Button
												className="w-full"
												variant="outline"
												onClick={handleAddToCart}
												disabled={event.status === 'soldout' || event.availableSpots === 0}
											>
												Add to Cart
											</Button>
										)}
										<div className="flex gap-2">
											<Button variant="outline" size="icon" className="flex-1">
												<Heart className="h-4 w-4" />
											</Button>
											<Button variant="outline" size="icon" className="flex-1" onClick={handleShare}>
												<Share2 className="h-4 w-4" />
											</Button>
										</div>
									</div>

									{event.availableSpots && event.availableSpots < 10 && (
										<p className="text-xs text-center text-orange-600 font-medium">
											Only {event.availableSpots} tickets remaining!
										</p>
									)}

									<p className="text-xs text-center text-muted-foreground">
										Free cancellation up to 48 hours before the event
									</p>
								</CardContent>
							</Card>
						)}
					</div>
				</div>

				{/* Sponsors Section */}
				{event.sponsors && event.sponsors.length > 0 && (
					<div className="mt-16">
						<div className="text-center mb-8">
							<h2 className="text-3xl font-bold mb-2">Event Sponsors</h2>
							<p className="text-muted-foreground">Thank you to our partners who make this event possible</p>
						</div>
						<InfiniteMovingCards
							items={event.sponsors.map(sponsor => ({
								name: sponsor.name,
								logoUrl: sponsor.logoUrl,
								link: sponsor.link
							}))}
							variant="logo"
							direction="left"
							speed="normal"
							pauseOnHover={true}
							className="mb-8"
						/>
					</div>
				)}
			</div>

			{/* Vehicle Submission Dialog */}
			<Dialog open={showSubmissionForm} onOpenChange={setShowSubmissionForm}>
				<DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
					<VehicleSubmissionForm
						eventId={event.id}
						eventSlug={event.slug}
						eventName={event.title}
						onSuccess={handleSubmissionSuccess}
						onCancel={() => setShowSubmissionForm(false)}
					/>
				</DialogContent>
			</Dialog>
		</div>
	);
}
