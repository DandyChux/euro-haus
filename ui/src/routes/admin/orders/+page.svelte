<script lang="ts">
	import { goto } from "$app/navigation";
	import apiClient from "$lib/api";
	import type { Fulfillment } from "./+page";

	let { data } = $props();
	let savingId = $state<string | null>(null);
	let actionError = $state("");
	let expandedId = $state<string | null>(null);

	const statuses = [
		"pending",
		"processing",
		"shipped",
		"delivered",
		"cancelled",
	];

	function changeStatus(event: Event) {
		const value = (event.currentTarget as HTMLSelectElement).value;
		void goto(
			value
				? `/admin/orders?status=${encodeURIComponent(value)}`
				: "/admin/orders",
		);
	}

	function formatDate(value?: string): string {
		if (!value) return "—";
		return new Date(value).toLocaleString([], {
			dateStyle: "medium",
			timeStyle: "short",
		});
	}

	function fulfillmentMethod(order: Fulfillment): "Shipping" | "Pickup" {
		return order.notes?.toLowerCase().includes("pickup")
			? "Pickup"
			: "Shipping";
	}

	function statusClass(status: string): string {
		if (status === "delivered")
			return "border-emerald-400/30 bg-emerald-400/10 text-emerald-200";
		if (status === "shipped")
			return "border-sky-400/30 bg-sky-400/10 text-sky-200";
		if (status === "cancelled")
			return "border-red-400/30 bg-red-400/10 text-red-200";
		if (status === "processing")
			return "border-amber-400/30 bg-amber-400/10 text-amber-200";
		return "border-white/10 bg-white/5";
	}

	async function updateOrder(order: Fulfillment, event: SubmitEvent) {
		event.preventDefault();
		const form = new FormData(event.currentTarget as HTMLFormElement);
		savingId = order.id;
		actionError = "";

		try {
			await apiClient.put(`/admin/fulfillments/${order.id}/status`, {
				status: String(form.get("status") ?? order.status),
				trackingNumber: String(form.get("tracking_number") ?? ""),
				trackingCarrier: String(form.get("tracking_carrier") ?? ""),
				notes: String(form.get("notes") ?? ""),
			});
			await goto(window.location.href, { invalidateAll: true });
		} catch (error) {
			actionError =
				error instanceof Error
					? error.message
					: "Unable to update order.";
		} finally {
			savingId = null;
		}
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
			Manage merchandise fulfillment from paid Stripe checkouts.
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
			{#each statuses as status}<option value={status}
					>{status[0].toUpperCase() + status.slice(1)}</option
				>{/each}
		</select>
	</label>
</header>

{#if data.error}
	<p
		class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
	>
		{data.error}
	</p>
{:else if actionError}
	<p
		class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
	>
		{actionError}
	</p>
{:else if data.orders.length === 0}
	<div
		class="rounded-3xl border border-dashed border-white/10 bg-white/5 p-8 text-sm"
	>
		No merchandise orders found.
	</div>
{:else}
	<div class="space-y-4">
		<div class="grid gap-3 sm:grid-cols-3">
			<div class="rounded-2xl border border-white/10 bg-white/5 p-4">
				<p
					class="text-xs uppercase tracking-wider text-muted-foreground"
				>
					Orders shown
				</p>
				<p class="mt-2 text-2xl font-semibold">{data.orders.length}</p>
			</div>
			<div class="rounded-2xl border border-white/10 bg-white/5 p-4">
				<p
					class="text-xs uppercase tracking-wider text-muted-foreground"
				>
					Needs action
				</p>
				<p class="mt-2 text-2xl font-semibold">
					{data.orders.filter((order) => order.status === "pending")
						.length}
				</p>
			</div>
			<div class="rounded-2xl border border-white/10 bg-white/5 p-4">
				<p
					class="text-xs uppercase tracking-wider text-muted-foreground"
				>
					Shipping / pickup
				</p>
				<p class="mt-2 text-2xl font-semibold">
					{data.orders.filter(
						(order) => fulfillmentMethod(order) === "Shipping",
					).length} / {data.orders.filter(
						(order) => fulfillmentMethod(order) === "Pickup",
					).length}
				</p>
			</div>
		</div>

		<div
			class="overflow-x-auto rounded-3xl border border-white/10 bg-white/5"
		>
			<table class="w-full min-w-[64rem] text-left text-sm">
				<thead
					class="border-b border-white/10 text-xs uppercase tracking-wider text-muted-foreground"
					><tr
						><th class="px-5 py-4">Product</th><th class="px-5 py-4"
							>Customer</th
						><th class="px-5 py-4">Method</th><th class="px-5 py-4"
							>Status</th
						><th class="px-5 py-4">Created</th><th class="px-5 py-4"
							>Actions</th
						></tr
					></thead
				>
				<tbody>
					{#each data.orders as order (order.id)}
						<tr
							class="border-b border-white/5 align-top last:border-0"
						>
							<td class="px-5 py-4"
								><div class="font-medium">
									{order.product_name || "Product"}
								</div>
								<div class="text-xs text-muted-foreground">
									Qty {order.quantity} · {order.type}
								</div></td
							>
							<td class="px-5 py-4"
								><div>{order.customer_name || "—"}</div>
								<div class="text-xs text-muted-foreground">
									{order.customer_email}
								</div></td
							>
							<td class="px-5 py-4"
								><span
									class="rounded-full border border-white/10 px-3 py-1 text-xs"
									>{fulfillmentMethod(order)}</span
								>{#if fulfillmentMethod(order) === "Shipping" && order.shipping_address}<div
										class="mt-2 max-w-48 whitespace-pre-line text-xs text-muted-foreground"
									>
										{order.shipping_address}
									</div>{/if}</td
							>
							<td class="px-5 py-4"
								><span
									class={`rounded-full border px-3 py-1 text-xs capitalize ${statusClass(order.status)}`}
									>{order.status}</span
								>{#if order.shipped_at}<div
										class="mt-2 text-xs text-muted-foreground"
									>
										Shipped {formatDate(order.shipped_at)}
									</div>{/if}{#if order.delivered_at}<div
										class="mt-1 text-xs text-muted-foreground"
									>
										Delivered {formatDate(
											order.delivered_at,
										)}
									</div>{/if}</td
							>
							<td
								class="whitespace-nowrap px-5 py-4 text-xs text-muted-foreground"
								>{formatDate(order.created_at)}
								<div class="mt-1 font-mono">
									{order.order_id}
								</div></td
							>
							<td class="px-5 py-4"
								><button
									class="rounded-xl border border-white/10 px-3 py-2 text-xs hover:border-white/30"
									onclick={() =>
										(expandedId =
											expandedId === order.id
												? null
												: order.id)}
									>{expandedId === order.id
										? "Close"
										: "Manage"}</button
								></td
							>
						</tr>
						{#if expandedId === order.id}
							<tr class="border-b border-white/5 bg-black/10"
								><td colspan="6" class="px-5 py-5"
									><form
										class="grid gap-4 lg:grid-cols-[12rem_12rem_1fr_auto]"
										onsubmit={(event) =>
											updateOrder(order, event)}
									>
										<label
											class="text-xs uppercase tracking-wider text-muted-foreground"
											>Status<select
												name="status"
												value={order.status}
												class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-3 py-2 text-sm text-foreground"
												>{#each statuses as status}<option
														value={status}
														>{status[0].toUpperCase() +
															status.slice(
																1,
															)}</option
													>{/each}</select
											></label
										><label
											class="text-xs uppercase tracking-wider text-muted-foreground"
											>Carrier<input
												name="tracking_carrier"
												value={order.tracking_carrier ??
													""}
												placeholder="USPS"
												class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-3 py-2 text-sm text-foreground"
											/></label
										><label
											class="text-xs uppercase tracking-wider text-muted-foreground"
											>Tracking number<input
												name="tracking_number"
												value={order.tracking_number ??
													""}
												placeholder="Tracking number"
												class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-3 py-2 text-sm text-foreground"
											/></label
										><button
											class="rounded-xl bg-white px-4 py-2 text-sm font-medium text-black disabled:opacity-50"
											disabled={savingId === order.id}
											>{savingId === order.id
												? "Saving…"
												: "Save"}</button
										><label
											class="text-xs uppercase tracking-wider text-muted-foreground lg:col-span-4"
											>Internal notes<textarea
												name="notes"
												rows="2"
												class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-3 py-2 text-sm text-foreground"
												>{order.notes ?? ""}</textarea
											></label
										>
									</form></td
								></tr
							>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	</div>
{/if}
