import React, { useState } from 'react';
import { Skeleton } from './skeleton';

interface ImageWithFallbackProps
	extends React.ImgHTMLAttributes<HTMLImageElement> {
	fallback?: React.ReactNode;
}

export const Image: React.FC<ImageWithFallbackProps> = ({
	fallback = <Skeleton className='w-full h-full' />,
	alt,
	...props
}) => {
	const [isLoading, setIsLoading] = useState(true);
	const [error, setError] = useState(false);

	return (
		<>
			{isLoading && fallback}
			<img
				{...props}
				alt={alt}
				onLoad={() => setIsLoading(false)}
				onError={() => {
					setIsLoading(false);
					setError(true);
				}}
				style={{ display: isLoading ? 'none' : 'block' }}
			/>
			{/* {error && <div className='text-destructive'>Failed to load image</div>} */}
		</>
	);
};
