<script lang="ts">
	import { onMount } from "svelte";
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import type { ProductVariant } from "$lib/schemas/product";

	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";
	import { Input } from "$lib/components/ui/input";

	interface Props {
		productId: string;
	}

	interface VariantResponse {
		variants?: ProductVariant[];
	}

	let { productId }: Props = $props();

	let variants = $state<ProductVariant[]>([]);
	let quantities = $state<Record<string, string>>({});

	let loading = $state(true);
	let saving = $state(false);
	let errorMessage = $state("");

	const totalStock = $derived(
		Object.values(quantities).reduce(
			(total, value) => total + (Number.parseInt(value, 10) || 0),
			0,
		),
	);

	const dirty = $derived(
		variants.some((variant) => {
			const originalQuantity =
				variant.stock_quantity === undefined
					? ""
					: String(variant.stock_quantity);

			return quantities[variant.id] !== originalQuantity;
		}),
	);

	onMount(() => {
		void loadVariants();
	});

	async function loadVariants() {
		loading = true;
		errorMessage = "";

		try {
			const response = await apiClient.get<VariantResponse>(
				`/admin/products/${productId}/variants`,
			);

			variants = response.variants ?? [];

			quantities = Object.fromEntries(
				variants.map((variant) => [
					variant.id,
					variant.stock_quantity === undefined
						? ""
						: String(variant.stock_quantity),
				]),
			);
		} catch (error) {
			console.error("Failed to load variant stock:", error);
			errorMessage = "Failed to load variant stock levels.";
		} finally {
			loading = false;
		}
	}

	function setQuantity(id: string, value: string) {
		if (value && !/^\d+$/.test(value)) {
			return;
		}

		quantities = {
			...quantities,
			[id]: value,
		};
	}

	function getStockStatus(quantity: string): string {
		const numericQuantity = Number.parseInt(quantity, 10) || 0;

		if (numericQuantity === 0) {
			return "Out of stock";
		}

		if (numericQuantity < 5) {
			return "Low stock";
		}

		return "In stock";
	}

	async function saveChanges() {
		if (!dirty || saving) {
			return;
		}

		saving = true;

		try {
			await apiClient.put(`/admin/products/${productId}/variants/stock`, {
				variants: variants.map((variant) => {
					const quantity = quantities[variant.id] ?? "";

					return {
						id: variant.id,
						stock_quantity:
							quantity === "" ? null : Number(quantity),
					};
				}),
			});

			toast.success("Stock levels updated.");

			await loadVariants();
		} catch (error) {
			console.error("Failed to update stock levels:", error);
			toast.error("Failed to update stock levels.");
		} finally {
			saving = false;
		}
	}
</script>

{#if loading}
	<div class="h-32 animate-pulse rounded-lg bg-muted"></div>
{:else if errorMessage}
	<Card class="p-5">
		<p class="text-sm text-destructive">
			{errorMessage}
		</p>

		<Button
			type="button"
			variant="outline"
			class="mt-4"
			onclick={loadVariants}
		>
			Retry
		</Button>
	</Card>
{:else if variants.length > 0}
	<Card class="space-y-5 p-5">
		<div>
			<h2 class="text-lg font-semibold">Variant stock levels</h2>

			<p class="text-sm text-muted-foreground">
				Inventory is stored in the database for each product variant.
			</p>
		</div>

		<div class="flex items-center justify-between rounded-lg bg-muted p-3">
			<span class="font-medium"> Total stock </span>

			<span class="rounded-full border px-2 py-1 text-sm">
				{totalStock} units
			</span>
		</div>

		<div class="space-y-3">
			{#each variants as variant (variant.id)}
				{@const quantity = quantities[variant.id] ?? ""}

				<div
					class="flex flex-col gap-3 rounded-lg border p-3 md:flex-row md:items-center"
				>
					<div class="min-w-0 flex-1">
						<p class="font-medium">
							{variant.variant || "Default"}
						</p>

						{#if variant.size || variant.color}
							<p class="text-sm text-muted-foreground">
								{variant.size ? `Size: ${variant.size}` : ""}

								{variant.size && variant.color ? " • " : ""}

								{variant.color ? `Color: ${variant.color}` : ""}
							</p>
						{/if}

						<p class="text-xs text-muted-foreground">
							Price: {(variant.price / 100).toFixed(2)}
						</p>
					</div>

					<div class="flex items-center gap-2">
						<Input
							type="number"
							min="0"
							value={quantity}
							placeholder="∞"
							class="w-24"
							onchange={(event) => {
								const input =
									event.currentTarget as HTMLInputElement;

								setQuantity(variant.id, input.value);
							}}
						/>

						<span class="rounded-full border px-2 py-1 text-xs">
							{getStockStatus(quantity)}
						</span>
					</div>
				</div>
			{/each}
		</div>

		{#if dirty}
			<p
				class="rounded-lg border border-yellow-300/50 bg-yellow-50 p-3 text-sm dark:bg-yellow-950/20"
			>
				You have unsaved stock changes.
			</p>
		{/if}

		<Button
			type="button"
			class="w-full"
			onclick={saveChanges}
			disabled={!dirty || saving}
		>
			{saving ? "Saving…" : "Save stock levels"}
		</Button>
	</Card>
{/if}
