<script lang="ts">
	import "./layout.css";
	import favicon from "$lib/assets/favicon.svg";
	import { page } from "$app/state";
	import { cart } from "$lib/stores/cart.svelte";
	import Navbar from "$lib/components/navbar.svelte";
	import Footer from "$lib/components/footer.svelte";
	import { Toaster } from "$lib/components/ui/sonner";

	let { children } = $props();

	const navLinks = [
		{ href: "/", label: "Home" },
		{ href: "/events", label: "Events" },
		{ href: "/catalog", label: "Catalog" },
		{ href: "/gallery", label: "Gallery" },
		{ href: "/about", label: "About" },
	];

	let isAdminRoute = $derived(page.url.pathname.startsWith("/admin"));
	let cartCount = $derived(
		cart.items.reduce((total, item) => total + item.quantity, 0),
	);
</script>

<svelte:head>
	<!-- <link rel="icon" href={favicon} /> -->
</svelte:head>

{#if isAdminRoute}
	<div class="min-h-screen">
		{@render children()}
	</div>
{:else}
	<div class="min-h-screen">
		<Navbar />
		<main>{@render children()}</main>
		<Footer />
		<Toaster position="top-center" richColors />
	</div>
{/if}
