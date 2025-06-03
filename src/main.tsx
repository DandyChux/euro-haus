import { RouterProvider, createRouter } from '@tanstack/react-router';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';

import { routeTree } from './routeTree.gen';
import { SearchProvider } from './lib/search-context';

const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
	interface Register {
		router: typeof router;
	}
}

const rootElement = document.getElementById('root')!;
if (!rootElement.innerHTML) {
	const root = createRoot(rootElement);
	root.render(
		<StrictMode>
			<SearchProvider>
				<RouterProvider router={router} />
			</SearchProvider>
		</StrictMode>
	);
}
