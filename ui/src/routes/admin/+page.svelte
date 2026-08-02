<script lang="ts">
	import { goto } from "$app/navigation";
	import { Button } from "$lib/components/ui/button";
	import * as Card from "$lib/components/ui/card";
	import {
		TriangleAlert,
		Calendar,
		Car,
		Package,
		Tag,
		FileImage,
	} from "@lucide/svelte";

	let { data } = $props();

	const quickActions = [
		{
			label: "Create Product",
			icon: Package,
			onClick: () => goto("/admin/products/new"),
		},
		{
			label: "Create Event",
			icon: Calendar,
			onClick: () => goto("/admin/events/new"),
		},
		{
			label: "Review Submissions",
			icon: Car,
			onClick: () => goto("/admin/submissions"),
		},
		{
			label: "Upload Media",
			icon: FileImage,
			onClick: () => goto("/admin/media", { state: { tab: "upload" } }),
		},
	];
</script>

<svelte:head>
	<title>Admin dashboard · Euro Haus</title>
</svelte:head>

<section class="space-y-6">
	{#if data.stats.error}
		<p
			class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
		>
			{data.stats.error}
		</p>
	{/if}

	<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
		<Card.Root class="rounded-3xl border p-5">
			<p class="text-sm">Products</p>
			<p class="mt-2 text-3xl font-semibold">
				{data.stats.isLoading ? "—" : data.stats.totalProducts}
			</p>
		</Card.Root>

		<Card.Root class="rounded-3xl border p-5">
			<p class="text-sm">Events</p>
			<p class="mt-2 text-3xl font-semibold">
				{data.stats.isLoading ? "—" : data.stats.totalEvents}
			</p>
		</Card.Root>

		<Card.Root class="rounded-3xl border p-5">
			<p class="text-sm">Featured items</p>
			<p class="mt-2 text-3xl font-semibold">
				{data.stats.isLoading ? "—" : data.stats.featuredItems}
			</p>
		</Card.Root>

		<Card.Root class="rounded-3xl border p-5">
			<p class="text-sm">Media files</p>
			<p class="mt-2 text-3xl font-semibold">
				{data.stats.isLoading ? "—" : data.stats.mediaFiles}
			</p>
		</Card.Root>

		<Card.Root class="rounded-3xl border p-5">
			<p class="text-sm">Pending submissions</p>
			<p class="mt-2 text-3xl font-semibold">
				{data.stats.isLoading ? "—" : data.stats.pendingSubmissions}
			</p>
		</Card.Root>

		<Card.Root class="rounded-3xl border p-5">
			<p class="text-sm">Active coupons</p>
			<p class="mt-2 text-3xl font-semibold">
				{data.stats.isLoading ? "—" : data.stats.activeCoupons}
			</p>
		</Card.Root>
	</div>

	<Card.Root class="mb-8">
		<Card.Header>
			<Card.Title>Quick Actions</Card.Title>
			<Card.Description>Common tasks and shortcuts</Card.Description>
		</Card.Header>
		<Card.Content>
			<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
				{#each quickActions as action}
					<Button
						variant="light"
						class="h-auto flex-col gap-2 p-4"
						onclick={action.onClick}
					>
						<action.icon class="h-5 w-5" />
						<span class="text-sm">{action.label}</span>
					</Button>
				{/each}
			</div>
		</Card.Content>
	</Card.Root>
</section>
