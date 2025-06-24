import { createRootRoute, Outlet } from '@tanstack/react-router';
import { Footer } from '~/components/footer';
import { Navbar } from '~/components/navbar';
import { Toaster } from 'sonner';

export const Route = createRootRoute({
	component: RootComponent,
});

function RootComponent() {
	return (
		<>
			<Navbar />
			<Outlet />
			<Footer />
			<Toaster
				position='top-center'
				richColors
			/>
		</>
	);
}
