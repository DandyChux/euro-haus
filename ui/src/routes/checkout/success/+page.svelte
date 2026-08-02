<script lang="ts">
	import { formatCurrency, formatDate } from "$lib/utils";
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Order successful · Euro Haus</title>
</svelte:head>

<section class="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
	<div class="rounded-3xl border border-white/10 bg-white/5 p-8">
		<h1 class="text-3xl font-semibold text-white">Order successful</h1>
		<p class="mt-4 text-base leading-7">
			Thanks for your purchase. Your order is in and processing has
			started.
		</p>

		<div
			class="mt-8 rounded-2xl border border-white/10 bg-black/20 p-5 text-sm"
		>
			<div class="flex items-center justify-between gap-4">
				<span>Order ID</span>
				<strong class="text-white">{data.order.id}</strong>
			</div>
			<div class="mt-3 flex items-center justify-between gap-4">
				<span>Total</span>
				<strong class="text-white"
					>{formatCurrency(data.order.amount / 100)}</strong
				>
			</div>
			<div class="mt-3 flex items-center justify-between gap-4">
				<span>Status</span>
				<strong class="text-white">{data.order.status}</strong>
			</div>
			{#if data.order.customer.email}
				<div class="mt-3 flex items-center justify-between gap-4">
					<span>Email</span>
					<strong class="text-white"
						>{data.order.customer.email}</strong
					>
				</div>
			{/if}
			<div class="mt-3 flex items-center justify-between gap-4">
				<span>Created</span>
				<strong class="text-white"
					>{formatDate(
						new Date(data.order.created * 1000).toISOString(),
					)}</strong
				>
			</div>
		</div>

		{#if data.order.items.length > 0}
			<div class="mt-8">
				<h2 class="text-xl font-medium text-white">Order summary</h2>
				<div
					class="mt-4 divide-y divide-white/10 rounded-2xl border border-white/10 bg-black/20"
				>
					{#each data.order.items as item (item.id)}
						<div
							class="flex items-center justify-between gap-4 px-5 py-4 text-sm"
						>
							<div>
								<p class="font-medium text-white">
									{item.name}
								</p>
								<p class="mt-1">Quantity: {item.quantity}</p>
							</div>
							<strong class="text-white"
								>{formatCurrency(item.amount / 100)}</strong
							>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<div class="mt-8 flex flex-wrap gap-3">
			<a
				href="/catalog"
				class="rounded-full bg-white px-5 py-3 text-sm font-medium"
				>Continue shopping</a
			>
			<a
				href="/"
				class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium text-white"
				>Return home</a
			>
		</div>
	</div>
</section>
