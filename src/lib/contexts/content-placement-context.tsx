import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react';
import { apiClient } from '../api';

interface DynamicPlacement {
	id: string;
	name: string;
	description: string;
	page: string;
	type: 'image' | 'video' | 'document';
	defaultUrl: string;
	currentUrl?: string;
}

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
						await apiClient.post('/content-placements/register', {
							id: p.id,
							name: p.name,
							description: p.description,
							page: p.page,
							type: p.type,
							mediaUrl: p.defaultUrl,
							currentUrl: p.currentUrl,
						});
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
		return placement?.currentUrl || defaultUrl;
	}, [activePlacements]);

	// Load saved placements from backend on mount
	useEffect(() => {
		apiClient.get('/content-placements/dynamic')
			.then(response => {
				const map = new Map<string, DynamicPlacement>();
				response.data.placements?.forEach((p: any) => {
					const placement: DynamicPlacement = {
						id: p.id,
						name: p.name,
						description: p.description,
						page: p.page,
						type: p.type,
						defaultUrl: p.mediaUrl,
						currentUrl: p.currentUrl || p.mediaUrl,
					};
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
