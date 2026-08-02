<script lang="ts">
	import { onMount } from "svelte";
	import { apiClient } from "$lib/api";
	import { formatDate } from "$lib/utils";

	interface Coupon {
		id: string;
		name: string;
		percent_off: number | null;
		amount_off: number | null;
		currency: string | null;
		duration: "once" | "repeating" | "forever";
		duration_in_months: number | null;
		max_redemptions: number | null;
		times_redeemed: number;
		valid: boolean;
		created: number;
	}

	let coupons = $state<Coupon[]>([]);
	let search = $state("");
	let statusFilter = $state<"all" | "active" | "expired">("all");
	let errorMessage = $state("");
	let statusMessage = $state("");
	let isLoading = $state(true);

	let form = $state({
		name: "",
		type: "percent" as "percent" | "fixed",
		value: "",
		duration: "once" as "once" | "repeating" | "forever",
		durationInMonths: "",
		maxRedemptions: "",
		currency: "usd",
	});

	let filteredCoupons = $derived.by(() => {
		return coupons.filter((coupon) => {
			if (statusFilter === "active" && !coupon.valid) return false;
			if (statusFilter === "expired" && coupon.valid) return false;
			if (!search.trim()) return true;
			return coupon.name.toLowerCase().includes(search.toLowerCase());
		});
	});

	function discountLabel(coupon: Coupon) {
		if (coupon.percent_off) return `${coupon.percent_off}% off`;
		if (coupon.amount_off)
			return `${(coupon.amount_off / 100).toFixed(2)} ${coupon.currency?.toUpperCase()} off`;
		return "Unknown";
	}

	async function loadCoupons() {
		isLoading = true;
		errorMessage = "";

		try {
			const response = await apiClient.get<{ coupons?: Coupon[] }>(
				"/admin/coupons",
			);
			coupons = response.coupons ?? [];
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to load coupons.";
		} finally {
			isLoading = false;
		}
	}

	async function createCoupon(event: SubmitEvent) {
		event.preventDefault();
		errorMessage = "";
		statusMessage = "";

		try {
			const payload: Record<string, unknown> = {
				name: form.name,
				duration: form.duration,
			};

			if (form.type === "percent") {
				payload.percent_off = Number.parseFloat(form.value);
			} else {
				payload.amount_off = Math.round(
					Number.parseFloat(form.value) * 100,
				);
				payload.currency = form.currency;
			}

			if (form.duration === "repeating" && form.durationInMonths) {
				payload.duration_in_months = Number.parseInt(
					form.durationInMonths,
					10,
				);
			}

			if (form.maxRedemptions) {
				payload.max_redemptions = Number.parseInt(
					form.maxRedemptions,
					10,
				);
			}

			await apiClient.post("/admin/coupons", payload);
			statusMessage = `Created coupon "${form.name}".`;

			form = {
				name: "",
				type: "percent",
				value: "",
				duration: "once",
				durationInMonths: "",
				maxRedemptions: "",
				currency: "usd",
			};

			await loadCoupons();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to create coupon.";
		}
	}

	async function createPromoCode(coupon: Coupon) {
		const raw = window.prompt(`Promo code for "${coupon.name}"`, "");
		const code = (raw ?? "")
			.trim()
			.toUpperCase()
			.replace(/[^A-Z0-9]/g, "");

		if (!code) return;

		try {
			await apiClient.post("/admin/promotion-codes", {
				coupon_id: coupon.id,
				code,
			});

			statusMessage = `Created promo code ${code}.`;
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to create promo code.";
		}
	}

	async function deleteCoupon(coupon: Coupon) {
		if (!window.confirm(`Delete "${coupon.name}"?`)) return;

		try {
			await apiClient.delete(`/admin/coupons/${coupon.id}`);
			statusMessage = `Deleted coupon "${coupon.name}".`;
			coupons = coupons.filter((entry) => entry.id !== coupon.id);
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to delete coupon.";
		}
	}

	onMount(() => {
		void loadCoupons();
	});
</script>

<svelte:head>
	<title>Admin coupons · Euro Haus</title>
</svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Coupons</h1>
		<p class="mt-3 max-w-3xl text-sm leading-7">
			Browse and delete coupons from the Svelte admin.
		</p>
	</div>
</header>

<section class="space-y-6">
	<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
		<form
			class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
			onsubmit={createCoupon}
		>
			<input
				bind:value={form.name}
				placeholder="Coupon name"
				class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
			/>

			<select
				bind:value={form.type}
				class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
			>
				<option value="percent">Percentage off</option>
				<option value="fixed">Fixed amount off</option>
			</select>

			<input
				bind:value={form.value}
				placeholder={form.type === "percent" ? "20" : "10.00"}
				class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
			/>

			<select
				bind:value={form.duration}
				class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
			>
				<option value="once">Once</option>
				<option value="forever">Forever</option>
				<option value="repeating">Repeating</option>
			</select>

			<input
				bind:value={form.durationInMonths}
				placeholder="Duration in months"
				disabled={form.duration !== "repeating"}
				class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3 disabled:opacity-40"
			/>

			<input
				bind:value={form.maxRedemptions}
				placeholder="Max redemptions"
				class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
			/>

			<button
				class="rounded-full bg-white px-5 py-3 text-sm font-medium md:col-span-2 xl:col-span-3"
			>
				Create coupon
			</button>
		</form>
	</div>

	<div
		class="grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-5 md:grid-cols-[minmax(0,1fr)_12rem]"
	>
		<input
			bind:value={search}
			placeholder="Search coupons"
			class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		/>

		<select
			bind:value={statusFilter}
			class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		>
			<option value="all">All statuses</option>
			<option value="active">Active</option>
			<option value="expired">Expired</option>
		</select>
	</div>

	{#if statusMessage}
		<p
			class="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-800"
		>
			{statusMessage}
		</p>
	{/if}

	{#if errorMessage}
		<p
			class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
		>
			{errorMessage}
		</p>
	{/if}

	{#if isLoading}
		<div class="rounded-3xl border border-white/10 bg-white/5 p-8 text-sm">
			Loading coupons…
		</div>
	{:else}
		<div class="space-y-4">
			{#each filteredCoupons as coupon (coupon.id)}
				<article
					class="rounded-3xl border border-white/10 bg-white/5 p-5"
				>
					<div
						class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"
					>
						<div>
							<div
								class="flex flex-wrap gap-2 text-xs uppercase tracking-[0.2em]"
							>
								<span
									>{coupon.valid ? "active" : "expired"}</span
								>
								<span>{coupon.duration}</span>
							</div>
							<h2 class="mt-2 text-lg font-medium">
								{coupon.name}
							</h2>
							<p class="mt-2 text-sm">
								{discountLabel(coupon)}
							</p>
							<p class="mt-2 text-sm">
								Created {formatDate(
									new Date(
										coupon.created * 1000,
									).toISOString(),
								)}
							</p>
							<p class="mt-1 text-sm">
								Redeemed {coupon.times_redeemed}
								{#if coupon.max_redemptions}
									/ {coupon.max_redemptions}{/if}
							</p>
						</div>

						<div class="flex flex-wrap gap-3">
							<button
								class="rounded-full border border-white/10 px-4 py-2 text-sm"
								onclick={() => void createPromoCode(coupon)}
							>
								Create promo code
							</button>
							<button
								class="rounded-full border border-destructive/30 px-4 py-2 text-sm text-destructive"
								onclick={() => void deleteCoupon(coupon)}
							>
								Delete
							</button>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
