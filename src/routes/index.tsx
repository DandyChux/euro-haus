import { createFileRoute, Link } from '@tanstack/react-router';
import { NewsletterSubscription } from '~/components/newsletter-subscription';
import { Button } from '~/components/ui/button';
import { SocialSection } from '~/components/social-section';
import EventCards from '~/components/event-cards';
import FeaturedProducts from '~/components/featured-products';
import { ManagedVideo } from '~/components/ui/managed-video';

export const Route = createFileRoute('/')({
	component: Index,
});

function Index() {

	return (
		<>
			{/* Hero Section */}
			<section className='w-full py-12 px-6 relative overflow-hidden'>
				<div className='absolute top-0 left-0 w-full h-full z-0'>
					<ManagedVideo
						src='https://euro-haus.nyc3.cdn.digitaloceanspaces.com/videos/euro-haus-intro.webm'
						name='Hero Background Video'
						description='Main hero section background video'
						autoPlay
						loop
						className='w-full h-full object-cover brightness-75'
					/>
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
								<a href='/#mailing-list'>Join the Community</a>
							</Button>
						</div>
					</div>
				</div>
			</section>

			{/* Upcoming Events */}
			<section className='w-full py-12 px-6 bg-muted'>
				<div className='max-w-7xl mx-auto'>
					<div className="flex items-center justify-between mb-8">
						<h2 className='text-3xl font-bold text-center'>
							Upcoming Events
						</h2>
						<Button asChild variant="outline">
							<Link to="/events">View All Events</Link>
						</Button>
					</div>

					<EventCards />
				</div>
			</section>

			<FeaturedProducts />

			{/* Featured Video */}
			<section className='w-full py-12 px-6'>
				<div className='max-w-5xl mx-auto'>
					<h2 className='text-3xl font-bold mb-8 text-center'>
						Featured Video
					</h2>

					<div className='rounded-3xl bg-muted text-muted-foreground shadow-neumorph p-6 overflow-hidden'>
						<div className='relative aspect-video rounded-2xl overflow-hidden shadow-neumorph-inset'>
							<ManagedVideo
								src='https://euro-haus.nyc3.cdn.digitaloceanspaces.com/videos/tonysporsche%5B1%5D.webm'
								name='Featured Video'
								description='Featured video showcase'
								controls
							/>
						</div>
						<div className='mt-6 text-center'>
							<h3 className='text-xl font-semibold mb-2'>
								Behind the Wheel: 1997 Porsche 911
							</h3>
							<p className='text-neutral-600 max-w-2xl mx-auto'>
								Experience the thrill of the Porsche 911 in action, showcasing the engineering marvel and pure driving experience that makes this vehicle a favorite among automotive enthusiasts.
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
