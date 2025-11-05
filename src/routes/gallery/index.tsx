import { createFileRoute, Link } from '@tanstack/react-router';
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';
import { Skeleton } from '~/components/ui/skeleton';
import { stripeService } from '~/lib/services/stripe-service';
import { galleryService } from '~/lib/services/gallery-service';
import type { EventGallery } from '~/lib/services/gallery-service';
import { Images, ArrowRight } from 'lucide-react';

export const Route = createFileRoute('/gallery/')({
	loader: async () => {
		// Fetch all events (including past ones)
		const events = await stripeService.getAllEvents(true);

		// Fetch galleries for all events
		const galleries = await galleryService.getAllEventGalleries(events);

		return { galleries };
	},
	pendingComponent: GalleryLoadingPage,
	component: GalleryIndexPage,
});

function GalleryLoadingPage() {
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
								<CardHeader>
									<Skeleton className="h-6 w-3/4 mb-2" />
									<Skeleton className="h-4 w-1/2" />
								</CardHeader>
								<CardContent>
									<div className="grid grid-cols-3 gap-2">
										<Skeleton className="aspect-square" />
										<Skeleton className="aspect-square" />
										<Skeleton className="aspect-square" />
									</div>
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
	);
}

function GalleryIndexPage() {
	const { galleries } = Route.useLoaderData();

	return (
		<div className="min-h-screen">
			{/* Hero Section */}
			<section className="bg-muted py-12 px-6">
				<div className="max-w-7xl mx-auto text-center">
					<div className="flex items-center justify-center gap-3 mb-4">
						<Images className="w-10 h-10" />
						<h1 className="text-4xl font-bold">Event Gallery</h1>
					</div>
					<p className="text-lg text-muted-foreground max-w-2xl mx-auto">
						Browse photos from our past events and see the incredible builds that have graced Euro Haus.
					</p>
				</div>
			</section>

			{/* Galleries Grid */}
			<section className="py-12 px-6">
				<div className="max-w-7xl mx-auto">
					{galleries.length === 0 ? (
						<div className="text-center py-12">
							<Images className="w-16 h-16 mx-auto mb-4 text-muted-foreground" />
							<p className="text-lg text-muted-foreground">No event galleries available yet. Check back soon!</p>
						</div>
					) : (
						<div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
							{galleries.map((gallery) => (
								<GalleryCard key={gallery.eventSlug} gallery={gallery} />
							))}
						</div>
					)}
				</div>
			</section>
		</div>
	);
}

function GalleryCard({ gallery }: { gallery: EventGallery }) {
	const previewImages = galleryService.getPreviewImages(gallery.images, 3);
	const totalImages = gallery.images.length;

	return (
		<Card className="overflow-hidden hover:shadow-lg transition-shadow">
			<CardHeader>
				<CardTitle className="text-xl">{gallery.eventName || gallery.eventSlug}</CardTitle>
				<p className="text-sm text-muted-foreground">
					{totalImages} {totalImages === 1 ? 'photo' : 'photos'}
				</p>
			</CardHeader>
			<CardContent>
				<div className="grid grid-cols-3 gap-2 mb-4">
					{previewImages.map((image, index) => (
						<div key={image.key} className="aspect-square overflow-hidden rounded-md">
							<Image
								src={image.url}
								alt={`Preview ${index + 1} from ${gallery.eventName || gallery.eventSlug}`}
								className="w-full h-full object-cover hover:scale-105 transition-transform duration-300"
							/>
						</div>
					))}
					{previewImages.length < 3 && Array.from({ length: 3 - previewImages.length }).map((_, i) => (
						<div key={`empty-${i}`} className="aspect-square bg-muted rounded-md" />
					))}
				</div>
			</CardContent>
			<CardFooter>
				<Link
					to="/gallery/$slug"
					params={{ slug: gallery.eventSlug }}
					className="w-full"
				>
					<Button className="w-full group">
						View All Photos
						<ArrowRight className="ml-2 w-4 h-4 group-hover:translate-x-1 transition-transform" />
					</Button>
				</Link>
			</CardFooter>
		</Card>
	);
}
