<script lang="ts">
	import { untrack } from "svelte";
	import { resolve } from "$app/paths";
	import Newsletter from "$lib/components/newsletter.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Label } from "$lib/components/ui/label";
	import * as RadioGroup from "$lib/components/ui/radio-group";
	import * as Select from "$lib/components/ui/select";
	import { formatDate } from "$lib/utils";
	import {
		getPriceAmount,
		getPriceName,
		priceIsSoldOut,
		type Price,
	} from "$lib/schemas/price";
	import apiClient from "$lib/api";
	import type { VehicleSubmission } from "$lib/schemas/event";
	import VehicleSubmissionForm from "$lib/components/vehicle-submission-form.svelte";

	let { data } = $props();

	type CheckoutState = "idle" | "submission" | "merchandise" | "loading";

	let checkoutState = $state<CheckoutState>("idle");
	let pendingPrice = $state<Price | null>(null);
	let quantity = $state("1");

	let selectedAddOns = $state<Array<{ price_id: string; quantity: number }>>(
		[],
	);

	let selectedPriceID = $state(
		untrack(
			() =>
				data.event.prices.find(
					(price) => price.id && !priceIsSoldOut(price),
				)?.id ??
				data.event.prices.find((price) => price.id && price.default)
					?.id ??
				data.event.prices.find((price) => price.id)?.id ??
				"",
		),
	);

	let selectedPrice = $derived(
		data.event.prices.find((price) => price.id === selectedPriceID),
	);

	function getSelectedPrice(): Price | undefined {
		return data.event.prices.find((price) => price.id === selectedPriceID);
	}

	function openCheckout(): void {
		const price = getSelectedPrice();

		console.log("CHECKOUT DEBUG", {
			selectedPriceID,
			price,
			requires_submission: price?.requires_submission,
			requiresSubmissionType: typeof price?.requires_submission,
		});

		if (!price?.id) {
			console.error("No ticket price selected", {
				selectedPriceID,
				prices: data.event.prices,
			});
			return;
		}

		// String(...) handles incorrectly serialized production values such as
		// "true" while remaining compatible with the correct boolean response.
		if (String(price.requires_submission) === "true") {
			pendingPrice = price;
			checkoutState = "submission";
			return;
		}

		void createStripeCheckout(price);
	}

	async function createStripeCheckout(price: Price): Promise<void> {
		if (data.event.status === "sold_out") {
			return;
		}

		checkoutState = "loading";

		try {
			const response = await apiClient.post<{
				url?: string;
				session_id?: string;
			}>("/create-event-checkout-session", {
				event_id: data.event.id,
				price_id: price.id,
				quantity: Number(quantity),
				addon_products: selectedAddOns,
			});

			if (!response.url) {
				throw new Error("Stripe Checkout URL was not returned");
			}

			window.location.assign(response.url);
		} catch (error) {
			console.error("Unable to create Stripe Checkout session", error);
			checkoutState = "idle";
		}
	}

	async function completeVehicleSubmission(
		submission: VehicleSubmission,
	): Promise<void> {
		if (!selectedPrice) {
			throw new Error("No ticket price selected");
		}

		checkoutState = "loading";

		const result = await apiClient.post<{
			session_id: string;
			session_url: string;
			requires_approval: boolean;
		}>("/create-participant-checkout", {
			submission_id: submission.id,
			price_id: selectedPrice.id,
			event_name: data.event.name,
			quantity: Number(quantity),
		});

		if (!result.session_url) {
			throw new Error("Checkout URL was not returned");
		}

		window.location.assign(result.session_url);
	}
</script>

<svelte:head>
	<title>{data.event.name} | Euro Haus</title>
	<meta
		name="description"
		content={`${data.event.description} View event details and purchase tickets.`}
	/>
</svelte:head>

<div id="top">
	<main>
		<section class="detail-hero">
			<div class="wrap detail-hero-copy">
				<a class="back-link" href={resolve("/events")}>← All events</a>
				<p class="eyebrow light">
					Event · {formatDate(data.event.date, {
						dateStyle: "short",
					})}
				</p>
				<h1>{data.event.name}</h1>
				<div class="detail-meta">
					<p>{formatDate(data.event.date)}</p>
					<p>{data.event.venue}</p>
					<p>{data.event.location}</p>
				</div>
			</div>
			<img
				src={data.event.images[0]}
				alt={`European cars at ${data.event.name}`}
			/>
		</section>

		<section class="detail-intro wrap section-pad">
			<div>
				<p class="section-label"><span>01</span> About the event</p>
			</div>
			<div>
				<h2>{data.event.description}</h2>
				<p>{data.event.long_description}</p>
			</div>
		</section>

		<section class="detail-grid wrap">
			<div class="detail-schedule">
				<p class="eyebrow">Schedule</p>
				{#each data.event.agenda as item (item.time)}
					<div>
						<time>{item.time}</time>
						<p>{item.activity}</p>
					</div>
				{/each}
			</div>
			<div class="detail-highlights">
				<p class="eyebrow">Included</p>
				<ul>
					{#each data.event.includes as highlight (highlight)}<li>
							{highlight}
						</li>{/each}
				</ul>
			</div>
		</section>

		{#if data.event.sponsors.length > 0}
			<section
				class="sponsors-section"
				aria-labelledby="sponsors-heading"
			>
				<div class="wrap sponsors-layout">
					<div class="sponsors-heading">
						<p class="eyebrow">Partners</p>
						<h2 id="sponsors-heading">
							Our<br /><em>sponsors.</em>
						</h2>
					</div>

					<div class="sponsors-grid">
						{#each data.event.sponsors as sponsor, index (`${sponsor.name}-${index}`)}
							<article class="sponsor-card">
								<div class="sponsor-logo">
									{#if sponsor.logo}
										<img
											src={sponsor.logo}
											alt={`${sponsor.name} logo`}
										/>
									{:else}
										<span aria-hidden="true">
											{sponsor.name.slice(0, 1)}
										</span>
									{/if}
								</div>

								<div class="sponsor-content">
									<p class="sponsor-tier">{sponsor.tier}</p>
									<h3>{sponsor.name}</h3>

									{#if sponsor.description}
										<p class="sponsor-description">
											{sponsor.description}
										</p>
									{/if}

									{#if sponsor.url}
										<a
											href={sponsor.url}
											target="_blank"
											rel="noreferrer"
										>
											Visit sponsor
											<span aria-hidden="true">↗</span>
										</a>
									{/if}
								</div>
							</article>
						{/each}
					</div>
				</div>
			</section>
		{/if}

		{#if checkoutState === "submission" && pendingPrice}
			<section class="submission-panel wrap" aria-live="polite">
				{#key pendingPrice.id}
					<VehicleSubmissionForm
						data={{ form: data.form }}
						eventId={data.event.id}
						priceId={pendingPrice.id}
						ticketTier={getPriceName(pendingPrice)}
						ticketPrice={getPriceAmount(pendingPrice)}
						ticketQuantity={Number(quantity)}
						requirements={pendingPrice.requirements}
						onsucceed={completeVehicleSubmission}
						oncancel={() => {
							pendingPrice = null;
							checkoutState = "idle";
						}}
					/>
				{/key}
			</section>
		{:else if checkoutState === "merchandise" && pendingPrice}
			<section class="checkout-panel wrap" aria-live="polite">
				<h2>Add event merchandise?</h2>
				<p>
					Choose any available merchandise before continuing to
					Stripe.
				</p>

				{#each data.linked_products as product (product.id)}
					<label class="merchandise-option">
						<input
							type="checkbox"
							onchange={(event) => {
								if (!product.price_id) {
									return;
								}

								if (event.currentTarget.checked) {
									selectedAddOns = [
										...selectedAddOns,
										{
											price_id: product.price_id,
											quantity: 1,
										},
									];
								} else {
									selectedAddOns = selectedAddOns.filter(
										(item) =>
											item.price_id !== product.price_id,
									);
								}
							}}
						/>

						<span>
							<strong>{product.title}</strong>
							<small>{product.description}</small>
						</span>
					</label>
				{/each}

				<div class="checkout-actions">
					<button
						type="button"
						onclick={() => {
							selectedAddOns = [];
							checkoutState = "idle";
						}}
					>
						Skip
					</button>

					<button
						type="button"
						onclick={() => {
							if (!pendingPrice) {
								return;
							}

							void createStripeCheckout(pendingPrice);
						}}
					>
						Continue to Stripe
					</button>
				</div>
			</section>
		{/if}

		<section class="ticket-section" id="tickets">
			<div class="wrap ticket-layout">
				<div class="ticket-heading">
					<p class="eyebrow light">Secure your place</p>
					<h2>Choose your<br /><em>ticket.</em></h2>
					<p>
						Payments are processed securely by Stripe. Ticket
						availability is limited by event capacity.
					</p>
				</div>
				<div class="ticket-panel">
					<RadioGroup.Root
						class="ticket-options"
						name="ticket-choice"
						bind:value={selectedPriceID}
					>
						{#each data.event.prices as price (price.id)}
							<Label
								class={`${selectedPriceID === price.id ? "active" : ""} ${
									priceIsSoldOut(price) ? "unavailable" : ""
								}`}
								for={`price-${price.id}`}
							>
								<RadioGroup.Item
									id={`price-${price.id}`}
									value={price.id}
									disabled={priceIsSoldOut(price)}
								/>

								<span>
									<strong>{getPriceName(price)}</strong>

									{#if price.description}
										<small>{price.description}</small>
									{/if}
								</span>

								<b>${getPriceAmount(price).toFixed(2)}</b>
							</Label>
						{/each}
					</RadioGroup.Root>
					<div>
						<input
							type="hidden"
							name="ticketId"
							value={selectedPriceID}
						/>
						<div class="quantity-field">
							<Label for="quantity">Quantity</Label><Select.Root
								type="single"
								name="quantity"
								bind:value={quantity}
								><Select.Trigger id="quantity"
									>{quantity}</Select.Trigger
								><Select.Content
									><Select.Group
										><Select.Label>Quantity</Select.Label
										>{#each [1, 2, 3, 4, 5, 6, 7, 8] as amount (amount)}<Select.Item
												value={String(amount)}
												label={String(amount)}
												>{amount}</Select.Item
											>{/each}</Select.Group
									></Select.Content
								></Select.Root
							>
						</div>
						<div class="ticket-total">
							<span>Total</span>
							<strong>
								${selectedPrice
									? (
											getPriceAmount(selectedPrice) *
											Number(quantity)
										).toFixed(2)
									: "0.00"}
							</strong>
						</div>
						<Button
							type="button"
							onclick={openCheckout}
							disabled={!getSelectedPrice() ||
								checkoutState === "loading" ||
								data.event.status === "sold_out"}
						>
							{checkoutState === "loading"
								? "Opening Stripe…"
								: "Continue to Stripe"}
							<span aria-hidden="true">↗</span>
						</Button>
						<p class="secure-note">
							Secure, hosted checkout. Prices shown in USD.
						</p>
					</div>
				</div>
			</div>
		</section>
		<Newsletter />
	</main>
</div>

<style>
	.detail-hero {
		padding-top: 110px;
		background: var(--foreground);
		color: var(--background);
	}
	.detail-hero-copy {
		padding-bottom: 65px;
	}
	.back-link {
		display: inline-block;
		margin-bottom: 65px;
		color: color-mix(in srgb, var(--background) 65%, var(--primary));
		font-size: 11px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.detail-hero h1 {
		max-width: 1200px;
		margin: 20px 0 55px;
		font-family: var(--font-display);
		font-size: clamp(100px, 12vw, 210px);
		font-weight: 800;
		letter-spacing: -0.06em;
		line-height: 0.72;
		text-transform: uppercase;
	}
	.detail-meta {
		display: flex;
		gap: 35px;
	}
	.detail-meta p {
		margin: 0;
		padding-left: 14px;
		border-left: 2px solid var(--accent);
		font-size: 12px;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.detail-hero > img {
		height: 68vh;
		min-height: 600px;
		filter: saturate(0.72);
	}
	.detail-intro {
		display: grid;
		grid-template-columns: 320px 1fr;
		gap: 70px;
	}
	.detail-intro h2 {
		max-width: 950px;
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(50px, 5.5vw, 92px);
		letter-spacing: -0.04em;
		line-height: 0.95;
		text-transform: uppercase;
	}
	.detail-intro > div:last-child > p {
		max-width: 730px;
		margin: 40px 0 0;
		color: var(--muted);
		font-size: 17px;
	}
	.detail-grid {
		display: grid;
		grid-template-columns: 1.1fr 0.9fr;
		padding-bottom: 130px;
	}
	.detail-schedule,
	.detail-highlights {
		padding: 55px;
		border: 1px solid var(--border);
	}
	.detail-highlights {
		border-left: 0;
	}
	.detail-schedule > div {
		display: grid;
		grid-template-columns: 100px 1fr;
		gap: 30px;
		padding: 20px 0;
		border-bottom: 1px solid var(--border);
	}
	.detail-schedule time {
		color: var(--primary);
		font-size: 11px;
		font-weight: 700;
	}
	.detail-schedule p {
		margin: 0;
	}
	.detail-highlights ul {
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.detail-highlights li {
		padding: 20px 0;
		border-bottom: 1px solid var(--border);
	}
	.sponsors-section {
		padding-block: 130px;
		background: var(--background);
	}

	.sponsors-layout {
		display: grid;
		grid-template-columns: 0.75fr 1.25fr;
		gap: 100px;
		align-items: start;
	}

	.sponsors-heading h2 {
		margin: 20px 0 0;
		font-family: var(--font-display);
		font-size: clamp(72px, 8vw, 145px);
		font-weight: 800;
		letter-spacing: -0.05em;
		line-height: 0.78;
		text-transform: uppercase;
	}

	.sponsors-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 16px;
	}

	.sponsor-card {
		display: flex;
		flex-direction: column;
		min-height: 320px;
		padding: 28px;
		border: 1px solid var(--border);
		transition:
			background-color 180ms ease,
			transform 180ms ease;
	}

	.sponsor-card:hover {
		background: color-mix(in srgb, var(--primary) 6%, var(--background));
		transform: translateY(-4px);
	}

	.sponsor-logo {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 130px;
		margin-bottom: 28px;
		background: var(--secondary);
		color: var(--background);
	}

	.sponsor-logo img {
		width: 100%;
		height: 100%;
		padding: 22px;
		object-fit: contain;
	}

	.sponsor-logo span {
		font-family: var(--font-display);
		font-size: 80px;
		font-weight: 800;
		line-height: 1;
		text-transform: uppercase;
	}

	.sponsor-content {
		display: flex;
		flex: 1;
		flex-direction: column;
		align-items: flex-start;
	}

	.sponsor-tier {
		margin: 0 0 8px;
		color: var(--primary);
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.14em;
		text-transform: uppercase;
	}

	.sponsor-card h3 {
		margin: 0;
		font-family: var(--font-display);
		font-size: 36px;
		font-weight: 700;
		letter-spacing: -0.03em;
		line-height: 0.9;
		text-transform: uppercase;
	}

	.sponsor-description {
		margin: 18px 0 0;
		color: var(--muted);
		font-size: 14px;
		line-height: 1.5;
	}

	.sponsor-card a {
		margin-top: auto;
		padding-top: 24px;
		color: var(--foreground);
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.sponsor-card a:hover {
		color: var(--primary);
	}

	.sponsor-card a span {
		margin-left: 6px;
		color: var(--primary);
	}
	.ticket-section {
		padding-block: 130px;
		background: var(--secondary);
		color: var(--background);
	}
	.ticket-layout {
		display: grid;
		grid-template-columns: 0.8fr 1.2fr;
		gap: 100px;
	}
	.ticket-heading h2 {
		margin: 25px 0;
		font-family: var(--font-display);
		font-size: clamp(80px, 8vw, 145px);
		letter-spacing: -0.05em;
		line-height: 0.78;
		text-transform: uppercase;
	}
	.ticket-panel {
		padding: 20px;
		background: var(--background);
		color: var(--foreground);
	}
	:global(.ticket-options [data-slot="label"]) {
		display: grid;
		grid-template-columns: 28px 1fr auto;
		gap: 18px;
		align-items: center;
		padding: 25px;
		border: 1px solid var(--border);
		border-bottom: 0;
		cursor: pointer;
	}
	:global(.ticket-options [data-slot="label"].active) {
		border-color: var(--primary);
		background: color-mix(in srgb, var(--primary) 8%, var(--background));
	}
	:global(.ticket-options [data-slot="label"].unavailable) {
		opacity: 0.45;
	}
	:global(.ticket-options [data-slot="label"] > span) {
		display: flex;
		flex-direction: column;
		gap: 5px;
	}
	:global(.ticket-options small),
	.secure-note {
		color: var(--muted);
	}
	:global(.ticket-options b) {
		font-family: var(--font-display);
		font-size: 28px;
	}
	.ticket-panel form {
		padding: 28px;
		border: 1px solid var(--border);
	}
	.quantity-field,
	.ticket-total {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.quantity-field :global([data-slot="select-trigger"]) {
		width: 96px;
	}
	.ticket-total {
		padding: 28px 0;
	}
	.ticket-total strong {
		font-family: var(--font-display);
		font-size: 48px;
	}
	.ticket-panel :global(.button) {
		width: 100%;
		border: 0;
		cursor: pointer;
	}
	.secure-note {
		font-size: 10px;
		text-align: center;
	}
	.submission-panel {
		padding-block: 100px;
		background:
			radial-gradient(
				circle at 10% 10%,
				color-mix(in srgb, var(--primary) 12%, transparent),
				transparent 35%
			),
			var(--secondary);
	}
	@media (max-width: 800px) {
		.detail-hero {
			padding-top: 70px;
		}
		.detail-hero h1 {
			font-size: 76px;
		}
		.detail-hero > img {
			height: 420px;
			min-height: 0;
		}
		.detail-meta {
			flex-direction: column;
			gap: 15px;
		}
		.detail-intro,
		.ticket-layout {
			display: block;
		}
		.detail-intro h2 {
			font-size: 52px;
		}
		.detail-grid {
			display: block;
			padding-bottom: 80px;
		}
		.detail-highlights {
			border-top: 0;
			border-left: 1px solid var(--border);
		}
		.sponsors-section {
			padding-block: 80px;
		}

		.sponsors-layout {
			display: block;
		}

		.sponsors-heading {
			margin-bottom: 50px;
		}

		.sponsors-heading h2 {
			font-size: 76px;
		}

		.sponsors-grid {
			grid-template-columns: 1fr;
		}

		.sponsor-card {
			min-height: 290px;
		}
		.ticket-section {
			padding-block: 80px;
		}
		.ticket-heading {
			margin-bottom: 50px;
		}
		.ticket-heading h2 {
			font-size: 72px;
		}
		.ticket-panel {
			padding: 10px;
		}
		:global(.ticket-options [data-slot="label"]) {
			padding: 20px 15px;
		}
		.submission-panel {
			padding-block: 50px;
		}
	}
</style>
