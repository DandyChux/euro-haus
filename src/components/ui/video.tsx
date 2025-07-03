import React from 'react';

export interface VideoProps
	extends React.VideoHTMLAttributes<HTMLVideoElement> {
	fallback?: React.ReactNode;
}

const createMp4Url = (baseUrl: string) => {
	return baseUrl.replace(/\.(mov|webm|avi|wmv|flv|mkv|3gp)$/i, '.mp4')
}

export const Video: React.FC<VideoProps> = ({
	...props
}) => {
	const baseUrl = props.src as string;

	return (
		<video
			// poster={fallback?.toString()}
			playsInline
			muted
			{...props}
		>
			<source src={baseUrl} type="video/webm" />
			<source src={createMp4Url(baseUrl)} type="video/mp4" />
			Your browser does not support the video tag.
		</video>
	);
};
