import { createFileRoute } from '@tanstack/react-router';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';
import { Link } from '@tanstack/react-router';
import { Calendar, MapPin, Users, Clock, ChevronLeft, Share2, Heart, Plus, Minus } from 'lucide-react';
import { Badge } from '~/components/ui/badge';
import { Separator } from '~/components/ui/separator';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Input } from '~/components/ui/input';
import { stripeService } from '~/lib/services/stripe-service';
import { useCart } from '~/lib/contexts/cart-context';
import { useState } from 'react';
import { toast } from 'sonner';
import { InfiniteMovingCards } from '~/components/ui/infinite-moving-cards';

export const Route = createFileRoute('/events/$slug')({
	loader: async ({ params }) => {
		const event = await stripeService.getEventBySlug(params.slug);
		if (!event) {
			throw new Error('Event not found');
		}
		return { event };
	},
	component: EventDetailPage,
});

function EventDetailPage() {
	const { event } = Route.useLoaderData();
	const { addItem } = useCart();
	const [quantity, setQuantity] = useState(1);

	const handleAddToCart = () => {
		if (event.status === 'soldout') {
			toast.error('This event is sold out');
			return;
		}

		if (event.availableSpots && quantity > event.availableSpots) {
			toast.error(`Only ${event.availableSpots} tickets remaining`);
			return;
		}

		addItem({
			id: event.id,
			priceId: event.priceId,
			title: `${event.title} - Ticket`,
			description: `Event on ${new Date(event.date).toLocaleDateString()}`,
			price: event.price,
			quantity,
			imageUrl: event.imageUrl,
			maxQuantity: event.availableSpots || event.maxQuantity,
			type: 'event',
			eventDate: event.date,
		});

		toast.success(`Added ${quantity} ticket${quantity > 1 ? 's' : ''} to cart`);
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
			// Fallback - copy to clipboard
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
										onClick={handleAddToCart}
										disabled={event.status === 'soldout' || event.availableSpots === 0}
									>
										{event.status === 'soldout' ? 'Sold Out' : 'Add to Cart'}
									</Button>
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
		</div>
	);
}
