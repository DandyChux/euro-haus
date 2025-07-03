import React, { useState, useEffect, useRef } from 'react';
import { cn } from '~/lib/utils';
import { Skeleton } from './skeleton';

export interface ImageProps
	extends React.ImgHTMLAttributes<HTMLImageElement> {
	fallback?: React.ReactNode;
}

export const Image: React.FC<ImageProps> = ({
	fallback = <Skeleton className='w-full h-full object-cover aspect-square' />,
	alt,
	src,
	className,
	...props
}) => {
	const [status, setStatus] = useState<'loading' | 'loaded' | 'error'>('loading');
	const imgRef = useRef<HTMLImageElement>(null);

	useEffect(() => {
		if (!src) {
			setStatus('error');
			return;
		}

		// Reset status when src changes
		setStatus('loading');

		// Create a new image element to preload
		const img = new window.Image();

		const handleLoad = () => {
			setStatus('loaded');
		};

		const handleError = () => {
			setStatus('error');
		};

		img.addEventListener('load', handleLoad);
		img.addEventListener('error', handleError);

		// Start loading
		img.src = src;

		// Check if image is already cached/loaded
		if (img.complete && img.naturalWidth > 0) {
			setStatus('loaded');
		}

		// Cleanup
		return () => {
			img.removeEventListener('load', handleLoad);
			img.removeEventListener('error', handleError);
		};
	}, [src]);

	return (
		<div className={cn("relative", className)}>
			{status !== 'loaded' && (
				<div className="absolute inset-0">
					{fallback}
				</div>
			)}
			<img
				ref={imgRef}
				{...props}
				src={src}
				alt={alt}
				className={cn(
					className,
					status !== 'loaded' && 'invisible'
				)}
				// Keep these as backup, but the main loading logic is in useEffect
				onLoad={() => setStatus('loaded')}
				onError={() => setStatus('error')}
			/>
		</div>
	);
};
