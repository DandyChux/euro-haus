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

/**
	* A custom hook that manages media content (images or videos) with content management capabilities.
	*
	* This hook provides a way to display media content that can be dynamically updated through
	* a content management system. It automatically registers the media placement with the content
	* management system and returns either the customized URL from the CMS or falls back to a default URL.
	*
	* @param options - Configuration options for the managed media
	* @param options.id - Optional custom ID for the placement. If not provided, one will be generated based on the current route and name
	* @param options.name - Display name for this media placement in the content management UI
	* @param options.description - Optional description of this media placement. Defaults to "[type] on [page name]"
	* @param options.type - The type of media content ('image' or 'video')
	* @param options.defaultUrl - The fallback URL to use if no custom content has been set
	*
	* @returns The URL to use for the media content (either the custom URL from the CMS or the default URL)
	*/
export function useManagedMedia(options: UseManagedMediaOptions) {
	const location = useLocation();
	const { registerPlacement } = useContentPlacementContext();
	const hasRegistered = useRef(false);
	const registrationKey = useRef('');
	const initialPathname = useRef(location.pathname);

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

	// Register this placement only when key changes AND we're on the same page where the component was initially rendered
	useEffect(() => {
		// Only register if we're still on the same page where this component was initially rendered
		if (location.pathname !== initialPathname.current) {
			return;
		}

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
	}, [currentKey, placementId, options.name, options.description, options.type, options.defaultUrl, pageName, savedPlacement?.mediaUrl, registerPlacement, location.pathname]);

	// Return the current URL (saved or default)
	return savedPlacement?.mediaUrl || options.defaultUrl;
}
