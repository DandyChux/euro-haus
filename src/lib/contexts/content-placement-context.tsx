import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react';
import { apiClient } from '../api';

interface BasePlacement {
	id: string;
	name: string;
	description: string;
	page: string;
}

interface MediaPlacement extends BasePlacement {
	type: 'image' | 'video' | 'document';
	defaultUrl: string;
	currentUrl?: string;
}

interface TextPlacement extends BasePlacement {
	type: 'text';
	defaultText: string;
	currentText?: string;
	html?: boolean;
}

type DynamicPlacement = MediaPlacement | TextPlacement;

interface ContentPlacementContextType {
	registerPlacement: (placement: DynamicPlacement) => void;
	getPlacementUrl: (id: string, defaultUrl: string) => string;
	activePlacements: Map<string, DynamicPlacement>;
}

const ContentPlacementContext = createContext<ContentPlacementContextType | null>(null);

export function ContentPlacementProvider({ children }: { children: React.ReactNode }) {
	const [activePlacements, setActivePlacements] = useState<Map<string, DynamicPlacement>>(new Map());
	const [isReady, setIsReady] = useState(false);
	const registeredIds = useRef(new Set<string>());
	const registrationTimeout = useRef<NodeJS.Timeout | null>(null);
	const pendingRegistrations = useRef<DynamicPlacement[]>([]);

	// Register a placement when a component mounts
	const registerPlacement = useCallback((placement: DynamicPlacement) => {
		// Skip if already registered
		if (registeredIds.current.has(placement.id)) {
			return;
		}

		registeredIds.current.add(placement.id);

		setActivePlacements(prev => {
			const newMap = new Map(prev);
			newMap.set(placement.id, placement);
			return newMap;
		});

		// Queue for batch registration
		pendingRegistrations.current.push(placement);

		// Clear existing timeout
		if (registrationTimeout.current) {
			clearTimeout(registrationTimeout.current);
		}

		// Set new timeout to batch registrations
		registrationTimeout.current = setTimeout(() => {
			if (pendingRegistrations.current.length > 0 && isReady) {
				// Register all pending placements
				pendingRegistrations.current.forEach(async (p) => {
					try {
						const basePayload = {
							id: p.id,
							name: p.name,
							description: p.description,
							page: p.page,
							type: p.type,
						};

						// Add type-specific properties based on placement type
						const payload = p.type === 'text'
							? {
								...basePayload,
								textContent: (p as TextPlacement).currentText || (p as TextPlacement).defaultText,
								defaultText: (p as TextPlacement).defaultText,
								html: (p as TextPlacement).html,
							}
							: {
								...basePayload,
								mediaUrl: (p as MediaPlacement).defaultUrl,
								currentUrl: (p as MediaPlacement).currentUrl,
							};

						await apiClient.post('/content-placements/register', payload);
					} catch (error) {
						console.error('Failed to register placement:', error);
					}
				});
				pendingRegistrations.current = [];
			}
		}, 500); // Wait 500ms to batch multiple registrations
	}, [isReady]);

	// Get the current URL for a placement
	const getPlacementUrl = useCallback((id: string, defaultUrl: string) => {
		const placement = activePlacements.get(id);
		if (!placement || placement.type === 'text') {
			return defaultUrl;
		}
		return placement.currentUrl || defaultUrl;
	}, [activePlacements]);

	// Load saved placements from backend on mount
	useEffect(() => {
		apiClient.get('/content-placements/dynamic')
			.then(response => {
				const map = new Map<string, DynamicPlacement>();
				response.data.placements?.forEach((p: {
					id: string;
					name: string;
					description: string;
					page: string;
					type: 'image' | 'video' | 'document' | 'text';
					mediaUrl?: string;
					currentUrl?: string;
					defaultText?: string;
					currentText?: string;
					html?: boolean;
				}) => {
					let placement: DynamicPlacement;

					if (p.type === 'text') {
						placement = {
							id: p.id,
							name: p.name,
							description: p.description,
							page: p.page,
							type: 'text',
							defaultText: p.defaultText || '',
							currentText: p.currentText,
							html: p.html
						};
					} else {
						placement = {
							id: p.id,
							name: p.name,
							description: p.description,
							page: p.page,
							type: p.type,
							defaultUrl: p.mediaUrl || '',
							currentUrl: p.currentUrl
						};
					}

					map.set(p.id, placement);
					registeredIds.current.add(p.id);
				});
				setActivePlacements(map);
				setIsReady(true);
			})
			.catch((error) => {
				console.error('Failed to load dynamic placements:', error);
				setIsReady(true);
			});

		// Cleanup timeout on unmount
		return () => {
			if (registrationTimeout.current) {
				clearTimeout(registrationTimeout.current);
			}
		};
	}, []);

	return (
		<ContentPlacementContext.Provider value={{ registerPlacement, getPlacementUrl, activePlacements }}>
			{children}
		</ContentPlacementContext.Provider>
	);
}

export function useContentPlacementContext() {
	const context = useContext(ContentPlacementContext);
	if (!context) {
		throw new Error('useContentPlacementContext must be used within ContentPlacementProvider');
	}
	return context;
}
