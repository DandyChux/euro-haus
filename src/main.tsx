import { RouterProvider, createRouter } from '@tanstack/react-router';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';

import { routeTree } from './routeTree.gen';
import { SearchProvider } from './lib/contexts/search-context';
import { CartProvider } from './lib/contexts/cart-context';
import { AuthProvider } from './lib/contexts/auth-context';

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
			<AuthProvider>
				<SearchProvider>
					<CartProvider>
						<RouterProvider router={router} />
					</CartProvider>
				</SearchProvider>
			</AuthProvider>
		</StrictMode>
	);
}
