import { createFileRoute } from '@tanstack/react-router';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';
import { Link } from '@tanstack/react-router';
import { Calendar, MapPin, Users, ChevronRight } from 'lucide-react';
import { Badge } from '~/components/ui/badge';
import { stripeService } from '~/lib/services/stripe-service';
import { Skeleton } from '~/components/ui/skeleton';

export const Route = createFileRoute('/events/')({
	loader: async () => {
		const events = await stripeService.getAllEvents()
		return { events }
	},
	pendingComponent: EventsLoadingPage,
	component: EventsPage,
});

function EventsLoadingPage() {
	return (
		<div className="min-h-screen">
			<section className="bg-muted py-12 px-6">
				<div className="max-w-7xl mx-auto text-center">
					<Skeleton className="h-10 w-64 mx-auto mb-4" />
					<Skeleton className="h-6 w-96 mx-auto" />
				</div>
			</section>
			<section className="py-12 px-6">
				<div className="max-w-7xl mx-auto">
					<div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
						{[1, 2, 3, 4, 5, 6].map((i) => (
							<Card key={i} className="overflow-hidden">
								<Skeleton className="aspect-video" />
								<CardHeader>
									<Skeleton className="h-6 w-3/4 mb-2" />
									<Skeleton className="h-4 w-full" />
									<Skeleton className="h-4 w-5/6" />
								</CardHeader>
								<CardContent>
									<Skeleton className="h-4 w-full mb-2" />
									<Skeleton className="h-6 w-24" />
								</CardContent>
								<CardFooter>
									<Skeleton className="h-10 w-full" />
								</CardFooter>
							</Card>
						))}
					</div>
				</div>
			</section>
		</div>
	)
}

function EventsPage() {
	const { events } = Route.useLoaderData()

	return (
		<div className="min-h-screen">
			{/* Hero Section */}
			<section className="bg-muted py-12 px-6">
				<div className="max-w-7xl mx-auto text-center">
					<h1 className="text-4xl font-bold mb-4">Upcoming Events</h1>
					<p className="text-lg text-muted-foreground max-w-2xl mx-auto">
						Join us for exclusive gatherings, track days, and technical workshops designed for true automotive enthusiasts.
					</p>
				</div>
			</section>

			{/* Events Grid */}
			<section className="py-12 px-6">
				<div className="max-w-7xl mx-auto">
					{events.length === 0 ? (
						<div className="text-center py-12">
							<p className="text-lg text-muted-foreground">No events scheduled at this time. Check back soon!</p>
						</div>
					) : (
						<div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
							{events.map((event) => (
								<Card key={event.slug} className="overflow-hidden hover:shadow-lg transition-shadow">
									<div className="aspect-video relative overflow-hidden">
										<Image
											src={event.imageUrl}
											alt={event.title}
											className="object-cover w-full h-full hover:scale-105 transition-transform duration-300"
										/>
										{event.status && (
											<Badge
												className="absolute top-4 right-4"
												variant={event.status === 'upcoming' ? 'default' : event.status === 'soldout' ? 'destructive' : 'secondary'}
											>
												{event.status === 'soldout' ? 'Sold Out' : event.status}
											</Badge>
										)}
									</div>
									<CardHeader>
										<CardTitle className="text-xl">{event.title}</CardTitle>
										<CardDescription className="space-y-2">
											<div className="flex items-center gap-2 text-sm">
												<Calendar className="h-4 w-4" />
												<span>{new Date(event.date).toLocaleDateString('en-US', {
													weekday: 'long',
													year: 'numeric',
													month: 'long',
													day: 'numeric'
												})}</span>
											</div>
											<div className="flex items-center gap-2 text-sm">
												<MapPin className="h-4 w-4" />
												<span>{event.location}</span>
											</div>
											{event.availableSpots && event.capacity && (
												<div className="flex items-center gap-2 text-sm">
													<Users className="h-4 w-4" />
													<span>{event.availableSpots} of {event.capacity} spots available</span>
												</div>
											)}
										</CardDescription>
									</CardHeader>
									<CardContent>
										<p className="text-sm text-muted-foreground line-clamp-2">{event.description}</p>
										<p className="mt-4 text-lg font-semibold">From ${event.price} USD</p>
									</CardContent>
									<CardFooter>
										<Button
											asChild
											className="w-full group"
											disabled={event.status === 'soldout' || event.status === 'cancelled'}
										>
											<Link to="/events/$slug" params={{ slug: event.slug }}>
												{event.status === 'soldout' ? 'Sold Out' : 'View Details'}
												<ChevronRight className="ml-2 h-4 w-4 group-hover:translate-x-1 transition-transform" />
											</Link>
										</Button>
									</CardFooter>
								</Card>
							))}
						</div>
					)}
				</div>
			</section>
		</div>
	);
}
