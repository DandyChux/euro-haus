import { useEffect, useMemo, useRef } from 'react';
import { useLocation } from '@tanstack/react-router';
import { useContentPlacementContext } from '../contexts/content-placement-context';
import { useContentPlacement } from './use-content-placement';

interface UseManagedMediaOptions {
	id?: string;
	name: string;
	description?: string;
	type: 'image' | 'video';
	defaultUrl: string;
}

export function useManagedMedia(options: UseManagedMediaOptions) {
	const location = useLocation();
	const { registerPlacement } = useContentPlacementContext();
	const hasRegistered = useRef(false);
	const registrationKey = useRef('');

	// Generate a unique ID based on location and name if not provided
	const placementId = useMemo(() => {
		if (options.id) return options.id;

		// Create ID from pathname and name
		const pathSegments = location.pathname.split('/').filter(Boolean);
		const pageName = pathSegments.length > 0 ? pathSegments.join('-') : 'home';
		return `${pageName}-${options.name.toLowerCase().replace(/\s+/g, '-')}`;
	}, [options.id, options.name, location.pathname]);

	// Get the page name from the route
	const pageName = useMemo(() => {
		const pathSegments = location.pathname.split('/').filter(Boolean);
		if (pathSegments.length === 0) return 'Home';
		return pathSegments.map(s => s.charAt(0).toUpperCase() + s.slice(1)).join(' / ');
	}, [location.pathname]);

	// Fetch saved URL from backend
	const { data: savedPlacement } = useContentPlacement(placementId);

	// Create a stable key for registration
	const currentKey = `${placementId}-${options.name}-${options.type}-${options.defaultUrl}`;

	// Register this placement only when key changes
	useEffect(() => {
		if (currentKey !== registrationKey.current) {
			registrationKey.current = currentKey;
			registerPlacement({
				id: placementId,
				name: options.name,
				description: options.description || `${options.type} on ${pageName}`,
				page: pageName,
				type: options.type,
				defaultUrl: options.defaultUrl,
				currentUrl: savedPlacement?.mediaUrl,
			});
		}
	}, [currentKey, placementId, options.name, options.description, options.type, options.defaultUrl, pageName, savedPlacement?.mediaUrl, registerPlacement]);

	// Return the current URL (saved or default)
	return savedPlacement?.mediaUrl || options.defaultUrl;
}
