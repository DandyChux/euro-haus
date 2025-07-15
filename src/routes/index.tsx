import { createFileRoute, Link } from '@tanstack/react-router';
import { NewsletterSubscription } from '~/components/newsletter-subscription';
import { Button } from '~/components/ui/button';
import { SocialSection } from '~/components/social-section';
import EventCards from '~/components/event-cards';
import FeaturedProducts from '~/components/featured-products';
import { ManagedVideo } from '~/components/ui/managed-video';
import { useQuery } from '@tanstack/react-query';
import { useManagedText } from '~/lib/hooks/use-managed-text';

interface YouTubeVideo {
	id: string;
	title: string;
	description: string;
	thumbnailUrl: string;
}

// Function to fetch latest YouTube video
async function fetchLatestYouTubeVideo(channelId: string, apiKey: string): Promise<YouTubeVideo | null> {
	try {
		// First, get the uploads playlist ID
		const channelResponse = await fetch(
			`https://www.googleapis.com/youtube/v3/channels?part=contentDetails&id=${channelId}&key=${apiKey}`
		);

		if (!channelResponse.ok) {
			throw new Error('Failed to fetch channel data');
		}

		const channelData = await channelResponse.json();
		const uploadsPlaylistId = channelData.items[0]?.contentDetails?.relatedPlaylists?.uploads;

		if (!uploadsPlaylistId) {
			throw new Error('No uploads playlist found');
		}

		// Then, get the latest video from the uploads playlist
		const videosResponse = await fetch(
			`https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=${uploadsPlaylistId}&maxResults=1&key=${apiKey}`
		);

		if (!videosResponse.ok) {
			throw new Error('Failed to fetch videos');
		}

		const videosData = await videosResponse.json();
		const latestVideo = videosData.items[0];

		if (!latestVideo) {
			return null;
		}

		return {
			id: latestVideo.snippet.resourceId.videoId,
			title: latestVideo.snippet.title,
			description: latestVideo.snippet.description,
			thumbnailUrl: latestVideo.snippet.thumbnails.high.url,
		};
	} catch (error) {
		console.error('Error fetching YouTube video:', error);
		return null;
	}
}

export const Route = createFileRoute('/')({
	component: HomePage,
});

function HomePage() {
	const YOUTUBE_API_KEY = import.meta.env.VITE_YOUTUBE_API_KEY;
	const YOUTUBE_CHANNEL_ID = import.meta.env.VITE_YOUTUBE_CHANNEL_ID;

	const heroTitle = useManagedText({
		name: 'Hero Title',
		defaultText: 'Welcome to Euro Haus',
		description: 'Main heading in the hero section'
	})

	const heroTagline = useManagedText({
		name: 'Hero Tagline',
		defaultText: 'Come for the cars, stay for the people.',
		description: 'Tagline displayed below the main heading'
	})

	// Fetch latest YouTube video
	const { data: latestVideo, isLoading: videoLoading } = useQuery({
		queryKey: ['latestYouTubeVideo'],
		queryFn: () => fetchLatestYouTubeVideo(YOUTUBE_CHANNEL_ID, YOUTUBE_API_KEY),
		staleTime: 1000 * 60 * 60, // Cache for 1 hour
		enabled: !!YOUTUBE_API_KEY && !!YOUTUBE_CHANNEL_ID,
	});

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
							<h1 className='text-white'>{heroTitle}</h1>
							<p className='mx-auto max-w-[700px] text-lg md:text-xl text-white/90'>
								{heroTagline}
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
							{videoLoading ? (
								<div className='w-full h-full flex items-center justify-center bg-muted'>
									<p className='text-muted-foreground'>Loading latest video...</p>
								</div>
							) : latestVideo ? (
								<iframe
									src={`https://www.youtube.com/embed/${latestVideo.id}?rel=0&modestbranding=1`}
									title={latestVideo.title}
									frameBorder='0'
									allow='accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share'
									allowFullScreen
									className='absolute inset-0 w-full h-full'
								/>
							) : (
								<div className='w-full h-full flex items-center justify-center bg-muted'>
									<p className='text-muted-foreground'>No videos available</p>
								</div>
							)}
						</div>
						<div className='mt-6 text-center'>
							<h3 className='text-xl font-semibold mb-2'>
								{latestVideo?.title || 'Latest from The Euro Haus'}
							</h3>
							<p className='text-neutral-600 max-w-2xl mx-auto line-clamp-3'>
								{latestVideo?.description || 'Check out our latest content showcasing European automotive excellence.'}
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
