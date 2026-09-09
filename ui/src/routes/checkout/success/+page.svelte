<script lang="ts">
	import { Button, buttonVariants } from "$lib/components/ui/button";
	import { formatCurrency, formatDate } from "$lib/utils";
	import type { PageProps } from "./$types";
	import * as Card from "$lib/components/ui/card";

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Order successful · Euro Haus</title>
</svelte:head>

<section class="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
	<Card.Root class="p-8">
		<Card.Title class="text-3xl font-semibold text-secondary"
			>Order successful</Card.Title
		>
		<p class="mt-4 text-base leading-7">
			Thanks for your purchase. Your order is in and processing has
			started.
		</p>

		<Card.Content
			class="mt-8 rounded-2xl border border-white/10 bg-black/20 p-5 text-sm"
		>
			<div class="flex items-center justify-between gap-4">
				<span>Order ID</span>
				<strong class="text-secondary">{data.order.id}</strong>
			</div>
			<div class="mt-3 flex items-center justify-between gap-4">
				<span>Total</span>
				<strong class="text-secondary"
					>{formatCurrency(data.order.amount / 100)}</strong
				>
			</div>
			<div class="mt-3 flex items-center justify-between gap-4">
				<span>Status</span>
				<strong class="text-secondary">{data.order.status}</strong>
			</div>
			{#if data.order.customer.email}
				<div class="mt-3 flex items-center justify-between gap-4">
					<span>Email</span>
					<strong class="text-secondary"
						>{data.order.customer.email}</strong
					>
				</div>
			{/if}
			<div class="mt-3 flex items-center justify-between gap-4">
				<span>Created</span>
				<strong class="text-secondary"
					>{formatDate(
						new Date(data.order.created * 1000).toISOString(),
					)}</strong
				>
			</div>
		</Card.Content>

		{#if data.order.items.length > 0}
			<div class="mt-8">
				<h2 class="text-xl font-medium text-secondary">
					Order summary
				</h2>
				<div
					class="mt-4 divide-y divide-white/10 rounded-2xl border border-white/10 bg-black/20"
				>
					{#each data.order.items as item (item.id)}
						<div
							class="flex items-center justify-between gap-4 px-5 py-4 text-sm"
						>
							<div>
								<p class="font-medium text-secondary">
									{item.name}
								</p>
								<p class="mt-1">Quantity: {item.quantity}</p>
							</div>
							<strong class="text-secondary"
								>{formatCurrency(item.amount / 100)}</strong
							>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<div class="mt-8 flex flex-wrap gap-3">
			<!-- <a
				href="/catalog"
				class="rounded-full bg-white px-5 py-3 text-sm font-medium"
				>Continue shopping</a
			>
			<a
				href="/"
				class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium text-secondary"
				>Return home</a
			> -->
			<Button
				variant="link"
				href="/catalog"
				class={buttonVariants({ variant: "default" })}
			>
				Continue shopping
			</Button>
			<Button
				variant="link"
				href="/"
				class={buttonVariants({ variant: "light" })}
			>
				Return home
			</Button>
		</div>
	</Card.Root>
</section>
