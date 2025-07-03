import React from 'react';
import { Video, VideoProps } from './video';
import { useManagedMedia } from '~/lib/hooks/use-managed-media';

interface ManagedVideoProps extends Omit<VideoProps, 'src'> {
	src: string;
	name: string;
	description?: string;
	placementId?: string;
}

export const ManagedVideo: React.FC<ManagedVideoProps> = ({
	src,
	name,
	description,
	placementId,
	...props
}) => {
	const managedSrc = useManagedMedia({
		id: placementId,
		name,
		description,
		type: 'video',
		defaultUrl: src,
	});

	return <Video {...props} src={managedSrc} />;
};
