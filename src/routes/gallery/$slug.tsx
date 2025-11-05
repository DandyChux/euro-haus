import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { Image } from '~/components/ui/image';
import { Button } from '~/components/ui/button';
import { Skeleton } from '~/components/ui/skeleton';
import { stripeService } from '~/lib/services/stripe-service';
import { galleryService } from '~/lib/services/gallery-service';
import type { MediaFile } from '~/lib/services/gallery-service';
import type { EventProduct } from '~/lib/services/stripe-service';
import { ArrowLeft, Calendar, MapPin, Download } from 'lucide-react';
import { format } from 'date-fns';
import { useState } from 'react';

interface GalleryData {
	event: EventProduct | null;
	images: MediaFile[];
}

export const Route = createFileRoute('/gallery/$slug')({
	loader: async ({ params }): Promise<GalleryData> => {
		const { slug } = params;

		// Fetch the event details
		const event = await stripeService.getEventBySlug(slug);

		// Fetch gallery images for this event
		const images = await galleryService.getEventGallery(slug);

		return { event, images };
	},
	pendingComponent: GalleryDetailLoadingPage,
	component: GalleryDetailPage,
});

function GalleryDetailLoadingPage() {
	return (
		<div className="min-h-screen">
			<section className="bg-muted py-8 px-6">
				<div className="max-w-7xl mx-auto">
					<Skeleton className="h-10 w-32 mb-4" />
					<Skeleton className="h-10 w-64 mb-2" />
					<div className="flex gap-4">
						<Skeleton className="h-5 w-32" />
						<Skeleton className="h-5 w-40" />
					</div>
				</div>
			</section>
			<section className="py-12 px-6">
				<div className="max-w-7xl mx-auto">
					<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
						{[1, 2, 3, 4, 5, 6, 7, 8, 9].map((i) => (
							<Skeleton key={i} className="aspect-square" />
						))}
					</div>
				</div>
			</section>
		</div>
	);
}

function GalleryDetailPage() {
	const { event, images } = Route.useLoaderData();
	const navigate = useNavigate();
	const [selectedImage, setSelectedImage] = useState<MediaFile | null>(null);

	if (!event && images.length === 0) {
		return (
			<div className="min-h-screen flex items-center justify-center">
				<div className="text-center">
					<h2 className="text-2xl font-bold mb-4">Gallery Not Found</h2>
					<p className="text-muted-foreground mb-6">
						This event gallery doesn't exist or hasn't been created yet.
					</p>
					<Link to="/gallery">
						<Button>
							<ArrowLeft className="mr-2 w-4 h-4" />
							Back to Galleries
						</Button>
					</Link>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen">
			{/* Header Section */}
			<section className="bg-muted py-8 px-6">
				<div className="max-w-7xl mx-auto">
					<Link to="/gallery">
						<Button variant="ghost" className="mb-4">
							<ArrowLeft className="mr-2 w-4 h-4" />
							Back to Galleries
						</Button>
					</Link>

					<h1 className="text-4xl font-bold mb-2">
						{event?.title || 'Event Gallery'}
					</h1>

					{event && (
						<div className="flex flex-wrap gap-4 text-muted-foreground">
							{event.date && (
								<div className="flex items-center gap-2">
									<Calendar className="w-4 h-4" />
									<span>{format(new Date(event.date), 'MMMM d, yyyy')}</span>
								</div>
							)}
							{event.location && (
								<div className="flex items-center gap-2">
									<MapPin className="w-4 h-4" />
									<span>{event.location}</span>
								</div>
							)}
						</div>
					)}

					<p className="mt-4 text-muted-foreground">
						{images.length} {images.length === 1 ? 'photo' : 'photos'}
					</p>
				</div>
			</section>

			{/* Gallery Grid */}
			<section className="py-12 px-6">
				<div className="max-w-7xl mx-auto">
					{images.length === 0 ? (
						<div className="text-center py-12">
							<p className="text-lg text-muted-foreground">
								No photos available for this event yet.
							</p>
						</div>
					) : (
						<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
							{images.map((image) => (
								<div
									key={image.key}
									className="aspect-square overflow-hidden rounded-lg cursor-pointer group relative"
									onClick={() => setSelectedImage(image)}
								>
									<Image
										src={image.url}
										alt={`Photo from ${event?.title || 'event'}`}
										className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
									/>
									<div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors" />
								</div>
							))}
						</div>
					)}
				</div>
			</section>

			{/* Lightbox Modal */}
			{selectedImage && (
				<ImageLightbox
					image={selectedImage}
					images={images}
					onClose={() => setSelectedImage(null)}
					onNext={() => {
						const currentIndex = images.findIndex(img => img.key === selectedImage.key);
						const nextIndex = (currentIndex + 1) % images.length;
						setSelectedImage(images[nextIndex]);
					}}
					onPrevious={() => {
						const currentIndex = images.findIndex(img => img.key === selectedImage.key);
						const previousIndex = (currentIndex - 1 + images.length) % images.length;
						setSelectedImage(images[previousIndex]);
					}}
				/>
			)}
		</div>
	);
}

interface ImageLightboxProps {
	image: MediaFile;
	images: MediaFile[];
	onClose: () => void;
	onNext: () => void;
	onPrevious: () => void;
}

function ImageLightbox({ image, images, onClose, onNext, onPrevious }: ImageLightboxProps) {
	const currentIndex = images.findIndex(img => img.key === image.key);
	const isFirst = currentIndex === 0;
	const isLast = currentIndex === images.length - 1;

	return (
		<div
			className="fixed inset-0 z-50 bg-black/90 flex items-center justify-center p-4"
			onClick={onClose}
		>
			<div className="relative w-full h-full flex items-center justify-center">
				{/* Close button */}
				<button
					onClick={onClose}
					className="absolute top-4 right-4 text-white hover:text-gray-300 z-10"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						className="h-8 w-8"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>

				{/* Previous button */}
				{!isFirst && (
					<button
						onClick={(e) => {
							e.stopPropagation();
							onPrevious();
						}}
						className="absolute left-4 text-white hover:text-gray-300 z-10"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							className="h-12 w-12"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M15 19l-7-7 7-7"
							/>
						</svg>
					</button>
				)}

				{/* Next button */}
				{!isLast && (
					<button
						onClick={(e) => {
							e.stopPropagation();
							onNext();
						}}
						className="absolute right-4 text-white hover:text-gray-300 z-10"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							className="h-12 w-12"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 5l7 7-7 7"
							/>
						</svg>
					</button>
				)}

				{/* Image */}
				<img
					src={image.url}
					alt="Full size"
					className="max-w-full max-h-full object-contain"
					onClick={(e) => e.stopPropagation()}
				/>

				{/* Image info */}
				<div className="absolute bottom-4 left-4 right-4 text-white text-center">
					<p className="text-sm">
						{currentIndex + 1} of {images.length}
					</p>
				</div>

				{/* Download button */}
				<a
					href={image.url}
					download
					target="_blank"
					rel="noopener noreferrer"
					onClick={(e) => e.stopPropagation()}
					className="absolute top-4 left-4 text-white hover:text-gray-300 z-10 flex items-center gap-2"
				>
					<Download className="w-6 h-6" />
				</a>
			</div>
		</div>
	);
}
