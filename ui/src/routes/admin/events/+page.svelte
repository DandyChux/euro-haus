<script lang="ts">
	import { formatCurrency, formatDate } from "$lib/utils";
	import { CirclePlus } from "@lucide/svelte";
	import type { PageProps } from "./$types";
	import { buttonVariants } from "$lib/components/ui/button";

	let { data }: PageProps = $props();

	let showPast = $state(false);

	let visibleEvents = $derived.by(() => {
		return data.events.filter((event) => {
			const isPast =
				new Date(event.date) < new Date() ||
				event.status === "cancelled";
			return showPast ? isPast : !isPast;
		});
	});
</script>

<svelte:head>
	<title>Admin events · Euro Haus</title>
</svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Event management</h1>
		<p class="mt-3 max-w-3xl text-sm leading-7">
			Browse events and drill into attendee/tier details. QR scanning and
			richer event actions can be ported next.
		</p>
	</div>
</header>

<section class="space-y-6">
	<a href="/admin/events/new" class={buttonVariants({ variant: "circle" })}>
		<CirclePlus />
		New Event
	</a>
	<div class="flex gap-3">
		<button
			class={[
				"rounded-full px-4 py-2 text-sm",
				!showPast ? "bg-white " : "border border-white/10 ",
			]}
			onclick={() => (showPast = false)}
		>
			Upcoming
		</button>

		<button
			class={[
				"rounded-full px-4 py-2 text-sm",
				showPast ? "bg-white " : "border border-white/10 ",
			]}
			onclick={() => (showPast = true)}
		>
			Past / cancelled
		</button>
	</div>

	<div class="grid gap-4">
		{#each visibleEvents as event (event.id)}
			<article class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<div
					class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"
				>
					<div>
						<p class="text-xs uppercase tracking-[0.2em]">
							{event.status || "upcoming"}
						</p>
						<h2 class="mt-2 text-xl font-medium">
							{event.name}
						</h2>
						<p class="mt-2 text-sm">
							{formatDate(event.date)}
							· {event.venue || event.location}
						</p>
						<p class="mt-3 line-clamp-2 text-sm leading-6">
							{event.description}
						</p>
						<p class="mt-3 text-sm">
							{#if event.prices.length > 0}
								{formatCurrency(
									Math.min(
										...event.prices
											.filter((price) => price.active)
											.map(
												(price) =>
													price.unit_amount / 100,
											),
									),
								)}
							{:else}
								Price unavailable
							{/if}
						</p>
					</div>

					<div class="flex flex-wrap gap-3">
						<a
							href={`/admin/event-details?id=${encodeURIComponent(event.id)}`}
							class="rounded-full bg-white px-4 py-2 text-sm"
						>
							Manage event
						</a>

						<a
							href={`/event/${encodeURIComponent(event.id)}`}
							target="_blank"
							rel="noreferrer"
							class="rounded-full border border-white/10 px-4 py-2 text-sm"
						>
							View live
						</a>
					</div>
				</div>
			</article>
		{/each}
	</div>
</section>
