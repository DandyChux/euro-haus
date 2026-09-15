<script lang="ts">
	import { goto } from "$app/navigation";

	let { data } = $props();

	function changeStatus(event: Event) {
		const value = (event.currentTarget as HTMLSelectElement).value;
		void goto(
			value
				? `/admin/orders?status=${encodeURIComponent(value)}`
				: "/admin/orders",
		);
	}
</script>

<svelte:head><title>Orders · Euro Haus Admin</title></svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 sm:flex-row sm:items-end sm:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Orders</h1>
		<p class="mt-3 max-w-2xl text-sm leading-7">
			Track merchandise products from paid Stripe checkouts through
			fulfillment.
		</p>
	</div>
	<label class="text-sm"
		>Status
		<select
			value={data.status}
			onchange={changeStatus}
			class="mt-2 rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		>
			<option value="">All orders</option>
			<option value="pending">Pending</option>
			<option value="processing">Processing</option>
			<option value="shipped">Shipped</option>
			<option value="delivered">Delivered</option>
			<option value="cancelled">Cancelled</option>
		</select>
	</label>
</header>

{#if data.error}
	<p
		class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
	>
		{data.error}
	</p>
{:else if data.orders.length === 0}
	<div
		class="rounded-3xl border border-dashed border-white/10 bg-white/5 p-8 text-sm"
	>
		No merchandise orders found.
	</div>
{:else}
	<div class="overflow-x-auto rounded-3xl border border-white/10 bg-white/5">
		<table class="w-full min-w-[50rem] text-left text-sm">
			<thead
				class="border-b border-white/10 text-xs uppercase tracking-wider text-muted-foreground"
			>
				<tr
					><th class="px-5 py-4">Product</th><th class="px-5 py-4"
						>Customer</th
					><th class="px-5 py-4">Order</th><th class="px-5 py-4"
						>Status</th
					><th class="px-5 py-4">Created</th></tr
				>
			</thead>
			<tbody>
				{#each data.orders as order (order.id)}
					<tr class="border-b border-white/5 last:border-0">
						<td class="px-5 py-4"
							><div class="font-medium">
								{order.product_name || "Product"}
							</div>
							<div class="text-xs text-muted-foreground">
								Qty {order.quantity}
							</div></td
						>
						<td class="px-5 py-4"
							><div>{order.customer_name || "—"}</div>
							<div class="text-xs text-muted-foreground">
								{order.customer_email}
							</div></td
						>
						<td class="px-5 py-4 font-mono text-xs"
							>{order.order_id}</td
						>
						<td class="px-5 py-4"
							><span
								class="rounded-full border border-white/10 px-3 py-1 text-xs capitalize"
								>{order.status}</span
							></td
						>
						<td class="whitespace-nowrap px-5 py-4"
							>{new Date(
								order.created_at,
							).toLocaleDateString()}</td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
