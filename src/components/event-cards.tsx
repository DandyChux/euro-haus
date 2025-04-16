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

import GT4Photo from '~/assets/gt4 reel photo.jpg';
import BMWMeetupPhoto from '~/assets/PANA1638.jpg';
import AudiPhoto from '~/assets/IMG_5047.jpg';
import { ChevronRight } from 'lucide-react';
import { cn } from '~/lib/utils';

type EventProps = {
	title: string;
	date: string;
	description?: string;
	location: string;
	price: number;
	image: string;
}

type EventCardProps = EventProps & {
	index: number;
	className?: string;
}

const events: EventProps[] = [
	{
		title: 'Porsche Club Meetup',
		date: '2025-06-01',
		location: 'Euro Haus',
		price: 149.99,
		image:
			GT4Photo,
	},
	{
		title: 'BMW Club Meetup',
		date: '2025-06-01',
		location: 'Euro Haus',
		price: 149.99,
		image:
			BMWMeetupPhoto,
	},
	{
		title: 'Audi Club Meetup',
		date: '2025-06-01',
		location: 'Euro Haus',
		price: 149.99,
		image:
			AudiPhoto,
	},
];

export default function EventCards() {
	return (
		<div className='grid md:grid-cols-2 lg:grid-cols-3 mx-auto gap-4 p-2 md:p-8'>
			{events.map((event, index) => (
				<EventCard key={index} index={index} {...event} />
			))}
		</div>
	)
}

const EventCard: React.FC<EventCardProps> = ({
	index,
	title,
	description,
	location,
	price,
	image,
	className,
	date
}) => {
	return (
		<Card className={cn('relative group rounded-none p-4 border-none flex flex-col', {
			'bg-secondary/10 text-foreground': index % 2 === 0,
			// 'text-primary': index % 2 !== 0,
		}, className)}>
			<CardHeader
				className={cn('px-2 gap-0', {
					'order-2': index % 2 !== 0,
					'order-1': index % 2 === 0,
				})}
			>
				<CardTitle
					className={cn('font-normal text-xl lg:text-3xl tracking-wide my-2', {
						'order-1': index % 2 === 0,
						'order-2': index % 2 !== 0,
					})}
				>
					{title}
				</CardTitle>
				<CardDescription
					className={cn('font-normal text-sm lg:text-base tracking-wide my-2 flex flex-col', {
						'order-1': index % 2 !== 0,
						'order-2': index % 2 === 0,
					})}
				>
					<span>{date}</span>
					<span>{description}</span>
					<span>From ${price} USD</span>
				</CardDescription>
			</CardHeader>
			<CardContent
				className={cn('p-0 flex-1 relative w-full aspect-[16/9] overflow-hidden', {
					'order-1': index % 2 !== 0,
					'order-2': index % 2 === 0,
				})}
			>
				<Image
					src={image}
					alt={title}
					className='absolute object-cover w-full h-full group-hover:scale-105 transition-transform duration-300'
				/>
			</CardContent>
			<CardFooter className='order-3'>
				<Button className='mt-4 group w-full flex items-center gap-1'>
					View Details
					<ChevronRight className='h-4 w-4 group-hover:translate-x-1 transition-transform' />
				</Button>
			</CardFooter>
		</Card>
	)
}
