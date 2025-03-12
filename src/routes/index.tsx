import { createFileRoute } from '@tanstack/react-router';
import { ChevronRight, Play } from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';

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
			<section className='w-full py-12 px-6'>
				<div className='max-w-7xl mx-auto'>
					<div className='rounded-3xl bg-neutral-100 shadow-neumorph p-6 overflow-hidden'>
						<div className='relative aspect-[16/9] rounded-2xl overflow-hidden shadow-neumorph-inset'>
							<Image
								src='https://sjc.microlink.io/Yq5ofAAPGt4bNWKW0734GeHqeBt0mXxKjnOP8BlRYRQ18wyUo2cuA9MQZKFtiaigsv49fo0U8vP6oBJH0SwhJQ.jpeg'
								alt='Porsche sports car'
								className='absolute object-cover w-full h-full'
							/>
							<div className='absolute inset-0 bg-gradient-to-r from-black/30 to-transparent flex items-center'>
								<div className='p-12'>
									<h1 className='text-4xl md:text-5xl font-bold mb-4'>
										European Auto <br />
										Excellence
									</h1>
									<p className='text-xl text-foreground/90 mb-8 max-w-md'>
										Specialized service and performance upgrades for premium
										European vehicles
									</p>
									<div className='flex flex-col sm:flex-row gap-4'>
										<Button className='shadow-neumorph-button hover:shadow-neumorph-button-hover'>
											View Services
											<ChevronRight className='h-4 w-4' />
										</Button>
										<Button>
											Watch Video
											<Play className='h-4 w-4' />
										</Button>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</section>

			<section className='w-full py-12 px-6'>
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
								<h3 className='text-xl font-semibold text-neutral-800 mb-2'>
									{event.title}
								</h3>
								<p className='text-neutral-500 text-sm mb-2'>{event.date}</p>
								<p className='text-neutral-500 text-sm mb-2'>
									From ${event.price} USD
								</p>
								<button className='mt-4 text-neutral-800 font-medium flex items-center gap-1 group'>
									Learn more
									<ChevronRight className='h-4 w-4 group-hover:translate-x-1 transition-transform' />
								</button>
							</div>
						))}
					</div>
				</div>
			</section>

			{/* Testimonials */}
			<section className='w-full py-12 px-6 bg-neutral-100'>
				<div className='max-w-7xl mx-auto'>
					<h2 className='text-3xl font-bold mb-8 text-center'>
						What Our Clients Say
					</h2>

					<div className='rounded-3xl bg-neutral-100 shadow-neumorph p-8'>
						<div className='grid grid-cols-1 md:grid-cols-2 gap-8'>
							{[
								{
									quote:
										'Euro Haus events are the highlight of my year. Their annual rally brings together the best European cars and enthusiasts in an unforgettable experience.',
									author: 'Michael S.',
									car: 'BMW M3 Owner',
								},
								{
									quote:
										"I've been attending Euro Haus meetups for years. The community they've built and the quality of their merchandise keeps me coming back every time.",
									author: 'Sarah L.',
									car: 'Audi RS5 Owner',
								},
							].map((testimonial, index) => (
								<div
									key={index}
									className='rounded-2xl bg-neutral-100 shadow-neumorph-inset p-6'
								>
									<p className='mb-4 italic text-foreground/75'>
										"{testimonial.quote}"
									</p>
									<div className='flex items-center gap-3'>
										<div className='h-10 w-10 rounded-full bg-neutral-200 shadow-neumorph flex items-center justify-center font-bold'>
											{testimonial.author.charAt(0)}
										</div>
										<div>
											<p className='font-semibold'>{testimonial.author}</p>
											<p className='text-sm text-muted-foreground'>
												{testimonial.car}
											</p>
										</div>
									</div>
								</div>
							))}
						</div>
					</div>
				</div>
			</section>

			{/* Newsletter */}
			<section className='w-full py-12 px-6'>
				<div className='max-w-3xl mx-auto'>
					<div className='rounded-3xl bg-neutral-100 shadow-neumorph p-8 text-center'>
						<h2 className='text-2xl font-bold text-neutral-800 mb-3'>
							Stay Updated
						</h2>
						<p className='text-neutral-600 mb-6'>
							Subscribe to our newsletter for the latest news, events, and
							special offers
						</p>

						<div className='flex flex-col sm:flex-row gap-3 max-w-md mx-auto'>
							<div className='flex-1 rounded-xl bg-neutral-100 shadow-neumorph-inset p-1'>
								<input
									type='email'
									placeholder='Your email address'
									className='w-full bg-transparent border-none outline-none px-3 py-2 text-neutral-800 placeholder:text-neutral-500'
								/>
							</div>
							<button className='px-6 py-3 rounded-xl bg-neutral-800 text-white font-medium hover:bg-neutral-700 transition-all duration-300'>
								Subscribe
							</button>
						</div>
					</div>
				</div>
			</section>

			{/* Featured Video */}
			<section className='w-full py-12 px-6'>
				<div className='max-w-5xl mx-auto'>
					<h2 className='text-3xl font-bold text-neutral-800 mb-8 text-center'>
						Featured Video
					</h2>

					<div className='rounded-3xl bg-neutral-100 shadow-neumorph p-6 overflow-hidden'>
						<div className='relative aspect-video rounded-2xl overflow-hidden shadow-neumorph-inset'>
							{/* This would be replaced with an actual video embed */}
							<div className='absolute inset-0 bg-neutral-200 flex items-center justify-center'>
								<div className='relative z-10 flex flex-col items-center'>
									<button className='h-20 w-20 rounded-full bg-neutral-100 shadow-neumorph flex items-center justify-center mb-4 hover:shadow-neumorph-hover transition-all duration-300'>
										<Play className='h-8 w-8 text-neutral-800 ml-1' />
									</button>
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
							<h3 className='text-xl font-semibold text-neutral-800 mb-2'>
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
		</>
	);
}
