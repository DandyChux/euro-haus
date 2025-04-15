import { createFileRoute, Link } from '@tanstack/react-router';
import { ChevronRight, Play } from 'lucide-react';
import { NewsletterSubscription } from '~/components/newsletter-subscription';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';
import EuroHausIntro from '~/assets/euro-haus-intro.mp4';
import { ProductCard } from '~/components/product-card';
import { SocialSection } from '~/components/social-section';

export const Route = createFileRoute('/')({
	component: Index,
});

const events = [
	{
		title: 'Porsche Club Meetup',
		date: '2025-06-01',
		location: 'Euro Haus',
		price: 149.99,
		image:
			'https://sjc.microlink.io/Yq5ofAAPGt4bNWKW0734GeHqeBt0mXxKjnOP8BlRYRQ18wyUo2cuA9MQZKFtiaigsv49fo0U8vP6oBJH0SwhJQ.jpeg',
	},
	{
		title: 'BMW Club Meetup',
		date: '2025-06-01',
		location: 'Euro Haus',
		price: 149.99,
		image:
			'https://sjc.microlink.io/Yq5ofAAPGt4bNWKW0734GeHqeBt0mXxKjnOP8BlRYRQ18wyUo2cuA9MQZKFtiaigsv49fo0U8vP6oBJH0SwhJQ.jpeg',
	},
	{
		title: 'Audi Club Meetup',
		date: '2025-06-01',
		location: 'Euro Haus',
		price: 149.99,
		image:
			'https://sjc.microlink.io/Yq5ofAAPGt4bNWKW0734GeHqeBt0mXxKjnOP8BlRYRQ18wyUo2cuA9MQZKFtiaigsv49fo0U8vP6oBJH0SwhJQ.jpeg',
	},
];

function Index() {
	return (
		<>
			{/* Hero Section */}
			<section className='w-full py-12 px-6 relative overflow-hidden'>
				<div className='absolute top-0 left-0 w-full h-full z-0'>
					<video className='w-full h-full object-cover brightness-75' autoPlay loop muted playsInline>
						<source
							src={EuroHausIntro}
							type='video/mp4'
						/>
					</video>
				</div>
				<div className='absolute inset-0 bg-black/50' />

				<div className='relative z-10 px-4 py-24 md:py-32 lg:py-40 md:px-6'>
					<div className='flex flex-col items-center space-y-4 text-center'>
						<div className='space-y-2'>
							<h1 className='text-white'>Welcome to Euro Haus</h1>
							<p className='mx-auto max-w-[700px] text-lg md:text-xl text-white/90'>
								Building a community around car enthusiasm. Join us for events, shop exclusive merchandise, and connect with fellow enthusiasts.
							</p>
						</div>
						<div className='inline-flex items-center space-x-4'>
							{/* <Button size='lg' asChild>
								<Link to='/'></Link>
							</Button> */}
							<Button size='lg' variant={'outline'} asChild>
								<Link to='/auth/register'>Join the Community</Link>
							</Button>
						</div>
					</div>
				</div>
			</section>

			{/* Upcoming Events */}
			<section className='w-full py-12 px-6 bg-muted'>
				<div className='max-w-7xl mx-auto'>
					<h2 className='text-3xl font-bold mb-8 text-center'>
						Upcoming Events
					</h2>

					<div className='grid grid-cols-1 md:grid-cols-3 gap-8'>
						{events.map((event, index) => (
							<div
								key={index}
								className='rounded-2xl bg-neutral-100 shadow-neumorph p-6 hover:shadow-neumorph-hover transition-all duration-300'
							>
								<div className='relative aspect-[16/9] rounded-2xl overflow-hidden shadow-neumorph-inset'>
									<Image
										src={event.image}
										alt={event.title}
										className='absolute object-cover w-full h-full'
									/>
								</div>
								<h3 className='text-xl font-semibold mb-2'>
									{event.title}
								</h3>
								<p className='text-neutral-500 text-sm mb-2'>{event.date}</p>
								<p className='text-neutral-500 text-sm mb-2'>
									From ${event.price} USD
								</p>
								<Button variant={'outline'} className='mt-4 font-medium flex items-center gap-1 group w-full'>
									View Details
									<ChevronRight className='h-4 w-4 group-hover:translate-x-1 transition-transform' />
								</Button>
							</div>
						))}
					</div>
				</div>
			</section>

			{/* Featured Products */}
			<section className="py-12 md:py-16 lg:py-20">
				<div className="px-4 md:px-6">
					<div className="flex flex-col items-center justify-center space-y-4 text-center">
						<div className="space-y-2">
							<h2 className="text-3xl font-bold tracking-tighter sm:text-4xl md:text-5xl">Featured Products</h2>
							<p className="max-w-[700px] text-muted-foreground md:text-xl/relaxed lg:text-base/relaxed xl:text-xl/relaxed">
								Shop our latest merchandise and exclusive Euro Haus gear.
							</p>
						</div>
					</div>
					<div className="mx-auto grid max-w-5xl grid-cols-1 gap-8 pt-12 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
						<ProductCard
							id='1'
							title="Euro Haus T-Shirt"
							description='Premium cotton t-shirt with Euro Haus logo'
							price={29.99}
							imageUrl="/placeholder.svg?height=400&width=400"
						/>
						<ProductCard
							id='2'
							title="Rally Cap"
							description='Adjustable cap with embroidered logo'
							price={24.99}
							imageUrl="/placeholder.svg?height=400&width=400"
						/>
						<ProductCard
							id='3'
							title="Orlando Rally 2025 Ticket"
							description='General admission to our flagship annual event'
							price={149.99}
							imageUrl="/placeholder.svg?height=400&width=400"
						/>
						<ProductCard
							id='9'
							title="Summer Car Meet Ticket"
							description='Entry to our summer gathering in Miami'
							price={49.99}
							imageUrl="/placeholder.svg?height=400&width=400"
						/>
					</div>
					<div className="mt-12 flex justify-center">
						<Button asChild>
							<Link to="/catalog">Shop All</Link>
						</Button>
					</div>
				</div>
			</section>

			{/* Featured Video */}
			<section className='w-full py-12 px-6'>
				<div className='max-w-5xl mx-auto'>
					<h2 className='text-3xl font-bold mb-8 text-center'>
						Featured Video
					</h2>

					<div className='rounded-3xl bg-muted text-muted-foreground shadow-neumorph p-6 overflow-hidden'>
						<div className='relative aspect-video rounded-2xl overflow-hidden shadow-neumorph-inset'>
							{/* This would be replaced with an actual video embed */}
							<div className='absolute inset-0 bg-neutral-200 flex items-center justify-center'>
								<div className='relative z-10 flex flex-col items-center'>
									<Button className='h-20 w-20 rounded-full shadow-neumorph flex items-center justify-center mb-4 hover:shadow-neumorph-hover transition-all duration-300 hover:bg-secondary'>
										<Play className='h-8 w-8 ml-1' />
									</Button>
									<p className='text-neutral-700 font-medium'>
										Project Spotlight: Porsche 911 Turbo Build
									</p>
								</div>
								<Image
									src='/placeholder.svg?height=720&width=1280&text=Video%20Thumbnail'
									alt='Video thumbnail'
									className='absolute w-full h-full object-cover'
								/>
							</div>
						</div>
						<div className='mt-6 text-center'>
							<h3 className='text-xl font-semibold mb-2'>
								Behind the Scenes: Complete Porsche Build
							</h3>
							<p className='text-neutral-600 max-w-2xl mx-auto'>
								Watch our team transform a stock Porsche 911 into a track-ready
								masterpiece with custom performance upgrades and meticulous
								attention to detail.
							</p>
						</div>
					</div>
				</div>
			</section>

			<NewsletterSubscription />

			<SocialSection />
		</>
	);
}
