import { createFileRoute } from '@tanstack/react-router';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';
import { Link } from '@tanstack/react-router';
import { Calendar, MapPin, Users, ChevronRight } from 'lucide-react';
import { Badge } from '~/components/ui/badge';
import { stripeService } from '~/lib/services/stripe-service';
import { Skeleton } from '~/components/ui/skeleton';
import { EventCard } from '~/components/event-cards';

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
								<EventCard event={event} />
							))}
						</div>
					)}
				</div>
			</section>
		</div>
	);
}
