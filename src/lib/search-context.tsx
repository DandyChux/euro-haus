import React, { createContext, useContext, useState, ReactNode } from 'react';

// Define the shape of our context
type SearchContextType = {
	searchQuery: string;
	setSearchQuery: (query: string) => void;
};

// Create the context with a default value
const SearchContext = createContext<SearchContextType | undefined>(undefined);

// Provider component that will wrap the application
export function SearchProvider({ children }: { children: ReactNode }) {
	const [searchQuery, setSearchQuery] = useState<string>('');

	return (
		<SearchContext.Provider value={{ searchQuery, setSearchQuery }}>
			{children}
		</SearchContext.Provider>
	);
}

// Custom hook to use the search context
export function useSearch() {
	const context = useContext(SearchContext);
	if (context === undefined) {
		throw new Error('useSearch must be used within a SearchProvider');
	}
	return context;
}
