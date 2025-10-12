import { useState, useEffect } from 'react';
import {
	Card,
	CardContent,
	CardHeader,
	CardTitle,
	CardFooter,
	CardDescription
} from './ui/card';
import { Image } from './ui/image';
import { Button } from './ui/button';
import { Skeleton } from './ui/skeleton';
import { cn } from '~/lib/utils';
import { Link } from '@tanstack/react-router';
import { Calendar, ChevronRight, MapPin, Users } from 'lucide-react';
import { stripeService } from '~/lib/services/stripe-service';
import type { EventProduct as Event } from '~/lib/services/stripe-service';
import { Badge } from './ui/badge';
import { useQuery } from '@tanstack/react-query';

type EventCardProps = {
	event: Event;
	className?: string;
}

// Loading skeleton component for event cards
function EventCardSkeleton({ index }: { index: number }) {
	return (
		<Card className={cn('relative rounded-none p-4 border-none flex flex-col', {
			'bg-secondary/10': index % 2 === 0,
		})}>
			<CardHeader className="px-2 gap-0">
				<Skeleton className="h-8 w-3/4 mb-2" />
				<div className="space-y-2">
					<Skeleton className="h-4 w-1/2" />
					<Skeleton className="h-4 w-full" />
					<Skeleton className="h-4 w-2/3" />
					<Skeleton className="h-4 w-1/3" />
				</div>
			</CardHeader>
			<CardContent className="p-0 flex-1 relative w-full min-h-[200px] max-h-[400px] overflow-hidden">
				<Skeleton className="absolute w-full h-full" />
			</CardContent>
			<CardFooter className="mt-4">
				<Skeleton className="h-10 w-full" />
			</CardFooter>
		</Card>
	);
}

export default function EventCards() {
	const { data: events, isLoading, error } = useQuery({
		queryKey: ['events'],
		queryFn: async () => {
			try {
				// Fetch featured events from Stripe
				const featuredEvents = await stripeService.getFeaturedEvents();
				return featuredEvents;
			} catch (err) {
				console.error('Failed to fetch events:', err);
				throw new Error('Failed to load events');
			}
		},
	});

	// Loading state
	if (isLoading) {
		return (
			<div className='grid md:grid-cols-2 lg:grid-cols-3 mx-auto gap-4 p-2 md:p-8'>
				{[0, 1, 2].map((index) => (
					<EventCardSkeleton key={index} index={index} />
				))}
			</div>
		);
	}

	// Error state
	if (error) {
		return (
			<div className='flex items-center justify-center p-8'>
				<div className='text-center'>
					<p className='text-muted-foreground mb-4'>{error.message}</p>
					<Button
						variant="outline"
						onClick={() => window.location.reload()}
					>
						Try Again
					</Button>
				</div>
			</div>
		);
	}

	// No events state
	if (events?.length === 0) {
		return (
			<div className='flex items-center justify-center p-8'>
				<div className='text-center'>
					<p className='text-lg text-muted-foreground mb-4'>
						No upcoming events at this time.
					</p>
					<Button asChild>
						<Link to="/events">View All Events</Link>
					</Button>
				</div>
			</div>
		);
	}

	return (
		<div className='grid md:grid-cols-2 lg:grid-cols-3 mx-auto gap-4 p-2 md:p-8'>
			{events?.map((event, index) => (
				<EventCard key={index} event={event} />
			))}
		</div>
	);
}

export const EventCard: React.FC<EventCardProps> = ({
	event,
	className
}) => {
	// Calculate the price display
	const getPriceDisplay = () => {
		// Use the new hasTiers and lowestPrice fields
		if (event.hasTiers && event.lowestPrice) {
			return `From $${event.lowestPrice.toFixed(2)} USD`;
		}
		// Regular single price
		return `$${event.price.toFixed(2)} USD`;
	};

	return (
		<Card className={cn("group overflow-hidden hover:shadow-lg transition-shadow", className)}>
			<div className="aspect-auto relative overflow-hidden">
				<Image
					src={event.images[0]}
					alt={event.title}
					className="object-contain w-full h-full"
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
							day: 'numeric',
							timeZone: 'UTC'
						})}</span>
					</div>
					<div className="flex items-center gap-2 text-sm">
						<MapPin className="h-4 w-4" />
						<span>{event.location}</span>
					</div>
					{/* {event.availableSpots && event.capacity && (
						<div className="flex items-center gap-2 text-sm">
							<Users className="h-4 w-4" />
							<span>{event.availableSpots} of {event.capacity} spots available</span>
						</div>
					)} */}
				</CardDescription>
			</CardHeader>
			<CardContent>
				<p className="text-sm text-muted-foreground line-clamp-2">{event.description}</p>
				<p className="mt-4 text-lg font-semibold">{getPriceDisplay()}</p>
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
	);
};
