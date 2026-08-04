<script lang="ts">
	import { formatCurrency, formatDate } from "$lib/utils";

	let { data } = $props();
</script>

<svelte:head>
	<title
		>{data.event
			? `${data.event.name} · Admin`
			: "Event details · Admin"}</title
	>
</svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Event Details</h1>
		<p class="mt-3 max-w-3xl text-sm leading-7">
			Manage the details of an event
		</p>
	</div>
</header>

<section class="space-y-6">
	{#if !data.event}
		<div
			class="rounded-3xl border border-dashed border-white/10 bg-white/5 p-8 text-sm"
		>
			Open this page with an `id` query string, for example:
			`/admin/event-details?id=0198b3c4-...`
		</div>
	{:else}
		<div class="grid gap-4 md:grid-cols-3">
			<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<p class="text-sm">Date</p>
				<p class="mt-2 font-medium">
					{formatDate(data.event.date)}
				</p>
			</div>

			<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<p class="text-sm">Venue</p>
				<p class="mt-2 font-medium">
					{data.event.venue || "—"}
				</p>
				<p class="mt-1 text-sm">{data.event.location}</p>
			</div>

			<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<p class="text-sm">Attendees</p>
				<p class="mt-2 font-medium">
					{data.attendees.length}
				</p>
			</div>
		</div>

		<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
			<h2 class="text-lg font-medium">Description</h2>

			<p class="mt-4 text-sm leading-7">
				{data.event.description}
			</p>

			{#if data.event.long_description}
				<h3 class="mt-6 text-sm font-medium uppercase tracking-wide">
					Long description
				</h3>

				<p class="mt-2 whitespace-pre-wrap text-sm leading-7">
					{data.event.long_description}
				</p>
			{/if}
		</div>

		<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
			<h2 class="text-lg font-medium">Prices</h2>

			{#if data.event.prices?.length === 0}
				<p class="mt-4 text-sm">
					This event uses a single default price.
				</p>
			{:else}
				<div class="mt-4 space-y-3">
					{#each data.event.prices as price (price.id)}
						<div
							class="rounded-2xl border border-white/10 bg-black/20 px-4 py-4"
						>
							<div
								class="flex items-center justify-between gap-4"
							>
								<div>
									<p class="font-medium">
										{price.nickname}
									</p>
									{#if price.description}
										<p class="mt-1 text-sm">
											{price.description}
										</p>
									{/if}
								</div>
								<strong class=""
									>{formatCurrency(
										price.unit_amount / 100,
									)}</strong
								>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
			<h2 class="text-lg font-medium">Linked products</h2>

			{#if data.linked_products.length === 0}
				<p class="mt-4 text-sm">No linked products found.</p>
			{:else}
				<div class="mt-4 space-y-3">
					{#each data.linked_products as product (product.id)}
						<a
							href={`/catalog/${product.id}`}
							target="_blank"
							rel="noreferrer"
							class="block rounded-2xl border border-white/10 px-4 py-3 text-sm hover:border-white/20 hover:text-primary"
						>
							{product.title} · {formatCurrency(product.price)}
						</a>
					{/each}
				</div>
			{/if}
		</div>

		<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
			<h2 class="text-lg font-medium">Attendees</h2>

			{#if data.attendees.length === 0}
				<p class="mt-4 text-sm">No attendees found.</p>
			{:else}
				<div
					class="mt-4 divide-y divide-white/10 rounded-2xl border border-white/10 bg-black/20"
				>
					{#each data.attendees as attendee (attendee.token)}
						<div
							class="grid gap-2 px-4 py-4 text-sm md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_10rem_8rem]"
						>
							<div>
								<p class="font-medium">
									{attendee.customerName ||
										"Unknown attendee"}
								</p>
								<p>{attendee.customerEmail || "—"}</p>
							</div>

							<div>{attendee.token || "—"}</div>

							<div>
								{attendee.ticketType || "General"}
							</div>

							<div>
								{attendee.checkedIn
									? "Checked in"
									: "Not checked in"}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</section>
