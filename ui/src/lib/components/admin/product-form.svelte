<script lang="ts">
	import { untrack } from "svelte";
	import { superForm, type SuperValidated } from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Textarea } from "$lib/components/ui/textarea";
	import { Button } from "$lib/components/ui/button";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Card } from "$lib/components/ui/card";

	import {
		ProductVariantsSchema,
		type ProductVariant,
		type ProductVariants,
	} from "$lib/schemas/product";
	import VariantForm from "./variant-form.svelte";
	import apiClient from "$lib/api";
	import { Label } from "../ui/label";

	interface Props {
		data: {
			form: SuperValidated<ProductVariants>;
		};

		onsaved: (data: ProductVariants) => Promise<void>;
	}

	let { data, onsaved }: Props = $props();

	const form = superForm(
		untrack(() => data.form),
		{
			SPA: true,
			validators: zod4Client(ProductVariantsSchema),

			async onUpdate({ form }) {
				if (!form.valid) return;

				try {
					await onsaved(form.data);
					toast.success("Product saved.");
				} catch (error) {
					console.error("Saving product failed:", error);
					toast.error("Unable to save product.");
				}
			},
		},
	);

	const { form: formData, enhance, submitting } = form;

	let hasVariants = $state($formData.variants.length > 0);
	let tagInput = $state("");
	let uploadingImage = $state(false);
	let uploadProgress = $state(0);
	let imageInput: HTMLInputElement;
	let imageUrl = $state("");
	let imageUrlError = $state("");

	interface UploadedMediaResponse {
		success: boolean;
		file: {
			key: string;
			url: string;
			last_modified: string;
			size: number;
			type: "image" | "video" | "other";
			folder: string;
		};
		message: string;
	}

	const isApparel = $derived($formData.category === "apparel");

	const categories = [
		{ value: "merchandise", label: "Merchandise" },
		{ value: "apparel", label: "Apparel" },
		{ value: "accessories", label: "Accessories" },
		{ value: "collectibles", label: "Collectibles" },
	];

	function addTag() {
		const tag = tagInput.trim();

		if (!tag || $formData.tags.includes(tag)) {
			return;
		}

		$formData.tags = [...$formData.tags, tag];
		tagInput = "";
	}

	function removeTag(index: number) {
		$formData.tags = $formData.tags.filter(
			(_, itemIndex) => itemIndex !== index,
		);
	}

	function productImageFolder(): string {
		const key = $formData.name
			.trim()
			.toLowerCase()
			.normalize("NFKD")
			.replace(/[^a-z0-9]+/g, "-")
			.replace(/(^-|-$)/g, "");

		return key ? `products/${key}/images` : "";
	}

	function addImageUrl() {
		const url = imageUrl.trim();

		if (!url) return;

		try {
			const parsedUrl = new URL(url);

			if (!["http:", "https:"].includes(parsedUrl.protocol)) {
				throw new Error("Unsupported URL protocol");
			}
		} catch {
			imageUrlError = "Enter a valid HTTP or HTTPS image URL.";
			return;
		}

		if ($formData.images.includes(url)) {
			imageUrlError = "This image has already been added.";
			return;
		}

		$formData.images = [...$formData.images, url];
		imageUrl = "";
		imageUrlError = "";
	}

	function removeImage(index: number) {
		$formData.images = $formData.images.filter(
			(_, imageIndex) => imageIndex !== index,
		);
	}

	function displayDollars(cents: number | null | undefined): number | string {
		if (cents == null || !Number.isFinite(cents)) {
			return "";
		}

		return cents / 100;
	}

	function setProductPrice(event: Event) {
		const rawValue = (event.currentTarget as HTMLInputElement).value;
		const dollars = Number(rawValue);

		$formData.price = Number.isFinite(dollars)
			? Math.round(Math.max(0, dollars) * 100)
			: 0;
	}

	function setCompareAtPrice(event: Event) {
		const rawValue = (event.currentTarget as HTMLInputElement).value;

		if (!rawValue.trim()) {
			$formData.compare_at_price = undefined;
			return;
		}

		const dollars = Number(rawValue);

		$formData.compare_at_price = Number.isFinite(dollars)
			? Math.round(Math.max(0, dollars) * 100)
			: undefined;
	}

	async function onFileUpload(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const selectedFiles = Array.from(input.files ?? []);

		if (selectedFiles.length === 0) {
			return;
		}

		const imageFiles = selectedFiles.filter((file) =>
			file.type.startsWith("image/"),
		);

		if (imageFiles.length !== selectedFiles.length) {
			toast.error(
				"Some files were skipped. Only image files are allowed.",
			);
		}

		if (imageFiles.length === 0) {
			input.value = "";
			return;
		}

		const folder = productImageFolder();

		if (!folder) {
			toast.error("Add a product title before uploading images.");
			input.value = "";
			return;
		}

		uploadingImage = true;
		uploadProgress = 0;

		const uploadedUrls: string[] = [];

		try {
			for (const [index, file] of imageFiles.entries()) {
				const body = new FormData();

				body.append("file", file, file.name);
				body.append("folder", folder);

				const response = await apiClient.upload<UploadedMediaResponse>(
					"/admin/media/upload",
					body,
					({ percent }) => {
						const completedFiles = index / imageFiles.length;

						const currentFileProgress =
							percent / 100 / imageFiles.length;

						uploadProgress = Math.round(
							(completedFiles + currentFileProgress) * 100,
						);
					},
				);

				uploadedUrls.push(response.file.url);
			}

			$formData.images = [...$formData.images, ...uploadedUrls];

			uploadProgress = 100;

			toast.success(
				`Uploaded ${uploadedUrls.length} image${
					uploadedUrls.length === 1 ? "" : "s"
				}.`,
			);
		} catch (error) {
			console.error("Failed to upload product images:", error);

			toast.error(
				uploadedUrls.length > 0
					? `Uploaded ${uploadedUrls.length} image${
							uploadedUrls.length === 1 ? "" : "s"
						}, but another upload failed.`
					: "Failed to upload product images.",
			);

			// Preserve files that uploaded before the failure.
			if (uploadedUrls.length > 0) {
				$formData.images = [...$formData.images, ...uploadedUrls];
			}
		} finally {
			uploadingImage = false;
			input.value = "";
		}
	}
</script>

<form method="POST" use:enhance class="space-y-6">
	<Card class="space-y-6 p-5">
		<div class="grid gap-4 md:grid-cols-2">
			<Form.Field {form} name="name">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Product title</Form.Label>

						<Input {...props} bind:value={$formData.name} />
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="category">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Product category</Form.Label>

						<select
							{...props}
							bind:value={$formData.category}
							class="w-full rounded-2xl border bg-transparent px-3 py-2"
						>
							{#each categories as category (category.value)}
								<option value={category.value}>
									{category.label}
								</option>
							{/each}
						</select>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>
		</div>

		<div class="grid gap-4 md:grid-cols-2">
			<Form.Field {form} name="price">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Price</Form.Label>

						<div class="flex items-center gap-2">
							<span class="text-sm text-muted-foreground">
								$
							</span>

							<Input
								{...props}
								type="number"
								inputmode="decimal"
								min="1"
								step="0.01"
								value={displayDollars($formData.price)}
								oninput={setProductPrice}
							/>
						</div>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="compare_at_price">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Compare-at price</Form.Label>

						<Input
							{...props}
							type="number"
							inputmode="decimal"
							min="1"
							step="0.01"
							value={displayDollars($formData.compare_at_price)}
							oninput={setCompareAtPrice}
						/>

						<Form.Description>
							Original price for sale items.
						</Form.Description>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>
		</div>

		<div class="grid gap-4 md:grid-cols-2">
			<Form.Field {form} name="subcategory">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Subcategory</Form.Label>

						<Input
							{...props}
							bind:value={$formData.subcategory}
							placeholder="T-Shirts, Wheels, Accessories..."
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="max_quantity">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Maximum quantity per order</Form.Label>

						<Input
							{...props}
							type="number"
							min="1"
							bind:value={$formData.max_quantity}
							placeholder="Leave blank for unlimited"
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>
		</div>

		<div class="flex flex-wrap gap-6 text-sm">
			<Form.Field {form} name="in_stock">
				<Form.Control>
					{#snippet children({ props })}
						<label class="flex items-center gap-2">
							<Checkbox
								{...props}
								bind:checked={$formData.in_stock}
							/>
							In stock
						</label>
					{/snippet}
				</Form.Control>
			</Form.Field>

			<Form.Field {form} name="is_new">
				<Form.Control>
					{#snippet children({ props })}
						<label class="flex items-center gap-2">
							<Checkbox
								{...props}
								bind:checked={$formData.is_new}
							/>
							New product
						</label>
					{/snippet}
				</Form.Control>
			</Form.Field>

			<Form.Field {form} name="featured">
				<Form.Control>
					{#snippet children({ props })}
						<label class="flex items-center gap-2">
							<Checkbox
								{...props}
								bind:checked={$formData.featured}
							/>

							Featured product
						</label>
					{/snippet}
				</Form.Control>
			</Form.Field>
		</div>

		<Form.Field {form} name="description">
			<Form.Control>
				{#snippet children({ props })}
					<Form.Label>Description</Form.Label>

					<Textarea {...props} bind:value={$formData.description} />
				{/snippet}
			</Form.Control>

			<Form.FieldErrors />
		</Form.Field>
	</Card>

	<Card class="space-y-4 p-5">
		<fieldset class="space-y-4">
			<legend class="text-sm font-medium"> Tags </legend>

			<div class="flex gap-2">
				<Input
					bind:value={tagInput}
					placeholder="Add a tag"
					onkeydown={(event) => {
						if (event.key === "Enter") {
							event.preventDefault();
							addTag();
						}
					}}
				/>

				<Button type="button" variant="outline" onclick={addTag}>
					Add
				</Button>
			</div>

			{#if $formData.tags.length > 0}
				<div class="flex flex-wrap gap-2">
					{#each $formData.tags as tag, index (tag)}
						<Button
							type="button"
							variant="secondary"
							size="sm"
							onclick={() => removeTag(index)}
						>
							{tag} ×
						</Button>
					{/each}
				</div>
			{/if}
		</fieldset>
	</Card>

	<Card class="space-y-4 p-5">
		<fieldset class="space-y-4">
			<legend class="text-sm font-medium"> Product images </legend>

			<div class="space-y-2">
				<label for="product-image-url" class="text-sm font-medium">
					Add an existing image URL
				</label>

				<div class="flex gap-2">
					<Input
						id="product-image-url"
						bind:value={imageUrl}
						type="url"
						placeholder="https://cdn.example.com/product-image.jpg"
						aria-invalid={imageUrlError ? "true" : undefined}
						onkeydown={(event) => {
							if (event.key === "Enter") {
								event.preventDefault();
								addImageUrl();
							}
						}}
					/>

					<Button type="button" onclick={addImageUrl}>
						Add image
					</Button>
				</div>

				{#if imageUrlError}
					<p class="text-sm text-destructive">
						{imageUrlError}
					</p>
				{/if}
			</div>

			<Label
				class="block cursor-pointer rounded-xl border border-dashed p-6 text-center"
				for="product-image-upload"
			>
				{#if uploadingImage}
					<p class="font-medium">
						Uploading images… {uploadProgress}%
					</p>
				{:else}
					<p class="font-medium">Choose product images</p>

					<p class="mt-1 text-sm text-muted-foreground">
						You can select multiple images. The first image is the
						default.
					</p>
				{/if}
			</Label>

			<input
				id="product-image-upload"
				bind:this={imageInput}
				class="sr-only"
				type="file"
				accept="image/*"
				multiple
				disabled={uploadingImage}
				onchange={onFileUpload}
			/>

			{#if $formData.images.length > 0}
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{#each $formData.images as image, index (image)}
						<div class="space-y-2 rounded-xl border p-3">
							<div
								class="relative aspect-video overflow-hidden rounded-lg bg-muted"
							>
								<img
									src={image}
									alt={`Product image ${index + 1}`}
									class="h-full w-full object-cover"
								/>

								{#if index === 0}
									<span
										class="absolute left-2 top-2 rounded bg-background/90 px-2 py-1 text-xs font-medium"
									>
										Default image
									</span>
								{/if}
							</div>

							<Button
								type="button"
								variant="ghost"
								size="sm"
								onclick={() => removeImage(index)}
							>
								Remove
							</Button>
						</div>
					{/each}
				</div>
			{/if}
		</fieldset>
	</Card>

	<Card class="space-y-5 p-5">
		<VariantForm {form} />
	</Card>

	<div class="flex justify-end border-t pt-4">
		<Button type="submit" disabled={$submitting}>
			{$submitting ? "Saving…" : "Save product"}
		</Button>
	</div>
</form>
