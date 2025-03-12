import { createRootRoute, Outlet } from '@tanstack/react-router';
import { Footer } from '~/components/footer';
import { Navbar } from '~/components/navbar';

export const Route = createRootRoute({
	component: RootComponent,
});

function RootComponent() {
	return (
		<>
			<Navbar />
			<Outlet />
			<Footer />
		</>
	);
}
