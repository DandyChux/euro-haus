import React from 'react';
import { Image, ImageProps } from './image';
import { useManagedMedia } from '~/lib/hooks/use-managed-media';

interface ManagedImageProps extends Omit<ImageProps, 'src'> {
	src: string;
	name: string;
	description?: string;
	placementId?: string;
}

export const ManagedImage: React.FC<ManagedImageProps> = ({
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
		type: 'image',
		defaultUrl: src,
	});

	return <Image {...props} src={managedSrc} />;
};
