import { useEffect, useMemo, useRef } from 'react';
import { useLocation } from '@tanstack/react-router';
import { useContentPlacementContext } from '../contexts/content-placement-context';
import { useContentPlacement } from './use-content-placement';

interface UseManagedTextOptions {
	id?: string;
	name: string;
	description?: string;
	defaultText: string;
	html?: boolean; // Whether the text contains HTML that should be rendered
}

/**
 * A custom hook that manages text content with content management capabilities.
 *
 * This hook provides a way to display text content that can be dynamically updated through
 * a content management system. It automatically registers the text placement with the content
 * management system and returns either the customized text from the CMS or falls back to default text.
 *
 * @param options - Configuration options for the managed text
 * @param options.id - Optional custom ID for the placement. If not provided, one will be generated based on the current route and name
 * @param options.name - Display name for this text placement in the content management UI
 * @param options.description - Optional description of this text placement. Defaults to "Text on [page name]"
 * @param options.defaultText - The fallback text to use if no custom content has been set
 * @param options.html - Whether the text contains HTML that should be rendered (defaults to false)
 *
 * @returns The text to display (either the custom text from the CMS or the default text)
 */
export function useManagedText(options: UseManagedTextOptions) {
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

	// Fetch saved text from backend
	const { data: savedPlacement } = useContentPlacement(placementId);

	// Create a stable key for registration
	const currentKey = `${placementId}-${options.name}-text-${options.defaultText}`;

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
				description: options.description || `Text on ${pageName}`,
				page: pageName,
				type: 'text',
				defaultText: options.defaultText,
				currentText: savedPlacement?.textContent,
				html: options.html || false,
			});
		}
	}, [
		currentKey,
		placementId,
		options.name,
		options.description,
		options.defaultText,
		options.html,
		pageName,
		savedPlacement?.textContent,
		registerPlacement,
		location.pathname
	]);

	// Return the current text (saved or default)
	return savedPlacement?.textContent || options.defaultText;
}
