<script lang="ts">
	import Newsletter from "$lib/components/newsletter.svelte";
	import { Badge } from "$lib/components/ui/badge";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Label } from "$lib/components/ui/label";

	const vehicles = [
		{
			make: "BMW",
			model: "E30 M3",
			years: "1988–1991",
			low: 65000,
			high: 145000,
			trend: "Rising",
			note: "Original, documented examples command a substantial premium.",
		},
		{
			make: "BMW",
			model: "E46 M3",
			years: "2001–2006",
			low: 24000,
			high: 72000,
			trend: "Rising",
			note: "Manual coupes lead the market; condition and subframe history matter.",
		},
		{
			make: "BMW",
			model: "E39 M5",
			years: "2000–2003",
			low: 32000,
			high: 85000,
			trend: "Rising",
			note: "Low-mile, stock cars with complete records remain most desirable.",
		},
		{
			make: "Porsche",
			model: "911 Carrera (997)",
			years: "2005–2012",
			low: 48000,
			high: 105000,
			trend: "Steady",
			note: "Generation, drivetrain, transmission, and service history drive value.",
		},
		{
			make: "Porsche",
			model: "Cayman S (987)",
			years: "2006–2012",
			low: 28000,
			high: 58000,
			trend: "Rising",
			note: "Manual 987.2 examples sit at the top of the range.",
		},
		{
			make: "Mercedes-Benz",
			model: "190E 2.3-16",
			years: "1986–1988",
			low: 24000,
			high: 72000,
			trend: "Rising",
			note: "Unmodified cars with healthy interiors are increasingly scarce.",
		},
		{
			make: "Mercedes-Benz",
			model: "E55 AMG (W211)",
			years: "2003–2006",
			low: 16000,
			high: 42000,
			trend: "Steady",
			note: "Maintenance records and suspension condition are critical.",
		},
		{
			make: "Audi",
			model: "B5 S4",
			years: "2000–2002",
			low: 12000,
			high: 38000,
			trend: "Rising",
			note: "Stock or thoughtfully modified six-speed cars lead demand.",
		},
		{
			make: "Audi",
			model: "R8 V8",
			years: "2008–2015",
			low: 68000,
			high: 125000,
			trend: "Steady",
			note: "Gated manual cars trade well above automated examples.",
		},
		{
			make: "Volkswagen",
			model: "Golf GTI Mk2",
			years: "1985–1992",
			low: 9000,
			high: 32000,
			trend: "Rising",
			note: "Original sixteen-valve cars attract the strongest interest.",
		},
		{
			make: "Volkswagen",
			model: "Corrado VR6",
			years: "1992–1995",
			low: 12000,
			high: 35000,
			trend: "Rising",
			note: "Clean bodywork and working active aero materially affect value.",
		},
		{
			make: "Volkswagen",
			model: "Golf R32 Mk4",
			years: "2004",
			low: 18000,
			high: 48000,
			trend: "Rising",
			note: "Original paint, low mileage, and complete history are key.",
		},
	];
	const makes = ["All", ...new Set(vehicles.map((vehicle) => vehicle.make))];
	let activeMake = $state("All");
	let query = $state("");
	let filtered = $derived(
		vehicles.filter(
			(vehicle) =>
				(activeMake === "All" || vehicle.make === activeMake) &&
				`${vehicle.make} ${vehicle.model}`
					.toLowerCase()
					.includes(query.toLowerCase().trim()),
		),
	);
	const currency = new Intl.NumberFormat("en-US", {
		style: "currency",
		currency: "USD",
		maximumFractionDigits: 0,
	});
</script>

<svelte:head
	><title>Pricing Guide | Euro Haus</title><meta
		name="description"
		content="Editorial market value ranges for popular European enthusiast cars from BMW, Porsche, Mercedes-Benz, Audi, and Volkswagen."
	/></svelte:head
>

<div id="top">
	<main>
		<section class="pricing-hero">
			<div class="wrap pricing-hero-grid">
				<div>
					<p class="eyebrow light">Enthusiast market · 2026</p>
					<h1>Know the<br /><em>market.</em></h1>
				</div>
				<div>
					<p>
						A clear starting point for shopping, selling, or simply
						understanding the cars we care about.
					</p>
					<span>Updated July 2026</span>
				</div>
			</div>
		</section>
		<section class="pricing-guide wrap section-pad">
			<div class="pricing-intro">
				<p class="section-label"><span>01</span> Vehicle guide</p>
				<div>
					<h2>
						Estimated market ranges for popular European enthusiast
						cars.
					</h2>
					<p>
						Figures reflect broad U.S. asking and transaction trends
						for running, presentable examples. Exceptional
						provenance, mileage, specification, and condition can
						move a car well beyond these ranges.
					</p>
				</div>
			</div>
			<div class="pricing-toolbar">
				<div class="make-filters" aria-label="Filter by make">
					{#each makes as make (make)}<Button
							variant={activeMake === make
								? "default"
								: "outline"}
							size="sm"
							onclick={() => (activeMake = make)}>{make}</Button
						>{/each}
				</div>
				<Label class="search-field"
					><span class="sr-only">Search vehicles</span><Input
						type="search"
						bind:value={query}
						placeholder="Search model"
					/><span aria-hidden="true">⌕</span></Label
				>
			</div>
			<div class="price-table" aria-live="polite">
				<div class="price-head">
					<span>Vehicle</span><span>Model years</span><span
						>Market range</span
					><span>Trend</span>
				</div>
				{#each filtered as vehicle (`${vehicle.make}-${vehicle.model}`)}<article
					>
						<div>
							<small>{vehicle.make}</small>
							<h3>{vehicle.model}</h3>
							<p>{vehicle.note}</p>
						</div>
						<span>{vehicle.years}</span><strong
							>{currency.format(vehicle.low)}–{currency.format(
								vehicle.high,
							)}</strong
						><Badge class="trend" variant="outline"
							>↗ {vehicle.trend}</Badge
						>
					</article>{:else}<div class="price-empty">
						<h3>No cars found.</h3>
						<p>Try another model or select a different make.</p>
					</div>{/each}
			</div>
		</section>
		<section class="pricing-method">
			<div class="wrap">
				<p class="eyebrow light">How to use this guide</p>
				<div class="method-grid">
					<article>
						<span>01</span>
						<h2>Start with condition.</h2>
						<p>
							Paint, interior, mechanical health, and evidence of
							careful ownership matter more than odometer alone.
						</p>
					</article>
					<article>
						<span>02</span>
						<h2>Value the history.</h2>
						<p>
							Service records, ownership documentation, original
							equipment, and known specialist work reduce
							uncertainty.
						</p>
					</article>
					<article>
						<span>03</span>
						<h2>Inspect the car.</h2>
						<p>
							Use these ranges as context—not a substitute for an
							independent pre-purchase inspection or professional
							appraisal.
						</p>
					</article>
				</div>
				<p class="pricing-disclaimer">
					Euro Haus does not provide financial advice or guaranteed
					valuations. Market conditions change, and all figures are
					editorial estimates in USD.
				</p>
			</div>
		</section>
		<Newsletter />
	</main>
</div>

<style>
	.pricing-hero {
		padding: 150px 0 110px;
		background: var(--secondary);
		color: var(--background);
	}
	.pricing-hero-grid {
		display: grid;
		grid-template-columns: 1fr 350px;
		gap: 80px;
		align-items: end;
	}
	.pricing-hero h1 {
		margin: 20px 0 0;
		font-family: var(--font-display);
		font-size: clamp(100px, 11vw, 195px);
		letter-spacing: -0.06em;
		line-height: 0.73;
		text-transform: uppercase;
	}
	.pricing-hero-grid > div:last-child p {
		margin: 0 0 35px;
		font-size: 17px;
	}
	.pricing-hero-grid span {
		font-size: 10px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.pricing-intro {
		display: grid;
		grid-template-columns: 320px 1fr;
		gap: 70px;
	}
	.pricing-intro h2 {
		max-width: 950px;
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(55px, 6vw, 100px);
		letter-spacing: -0.045em;
		line-height: 0.93;
		text-transform: uppercase;
	}
	.pricing-intro > div p {
		max-width: 720px;
		color: var(--muted);
	}
	.pricing-toolbar {
		display: flex;
		justify-content: space-between;
		gap: 30px;
		margin: 100px 0 35px;
		padding-bottom: 22px;
		border-bottom: 1px solid var(--border);
	}
	.make-filters {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.make-filters :global([data-slot="button"]) {
		min-width: 0;
	}
	.pricing-toolbar :global(.search-field) {
		position: relative;
		min-width: 250px;
	}
	.pricing-toolbar :global([data-slot="input"]) {
		width: 100%;
		height: auto;
		padding: 10px 30px 10px 0;
		border: 0;
		border-bottom: 1px solid var(--foreground);
		background: transparent;
		border-radius: 0;
		box-shadow: none;
	}
	.price-head,
	.price-table article {
		display: grid;
		grid-template-columns: 1.4fr 0.55fr 0.7fr 0.35fr;
		gap: 25px;
	}
	.price-head {
		padding: 14px 22px;
		background: var(--foreground);
		color: var(--background);
		font-size: 9px;
		text-transform: uppercase;
	}
	.price-table article {
		align-items: center;
		padding: 28px 22px;
		border: 1px solid var(--border);
		border-top: 0;
	}
	.price-table small,
	:global(.trend) {
		color: var(--primary);
		font-size: 9px;
		font-weight: 700;
		text-transform: uppercase;
	}
	.price-table h3 {
		margin: 4px 0;
		font-family: var(--font-display);
		font-size: 32px;
		text-transform: uppercase;
	}
	.price-table p {
		margin: 0;
		color: var(--muted);
		font-size: 11px;
	}
	.price-table strong {
		font-family: var(--font-display);
		font-size: 25px;
	}
	.price-empty {
		padding: 80px 20px;
		text-align: center;
	}
	.pricing-method {
		padding-block: 120px;
		background: var(--foreground);
		color: var(--background);
	}
	.method-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		margin-top: 60px;
		border: 1px solid
			color-mix(in srgb, var(--background) 20%, var(--foreground));
	}
	.method-grid article {
		min-height: 330px;
		padding: 35px;
		border-right: 1px solid
			color-mix(in srgb, var(--background) 20%, var(--foreground));
	}
	.method-grid h2 {
		margin: 105px 0 20px;
		font-family: var(--font-display);
		font-size: 38px;
		text-transform: uppercase;
	}
	.method-grid p,
	.pricing-disclaimer {
		color: color-mix(in srgb, var(--background) 60%, var(--primary));
	}
	.pricing-disclaimer {
		max-width: 780px;
		margin-top: 55px;
		font-size: 11px;
	}
	@media (max-width: 800px) {
		.pricing-hero {
			padding: 100px 0 75px;
		}
		.pricing-hero-grid,
		.pricing-intro {
			display: block;
		}
		.pricing-hero h1 {
			font-size: 72px;
		}
		.pricing-intro h2 {
			font-size: 52px;
		}
		.pricing-toolbar {
			display: block;
			margin-top: 65px;
		}
		.pricing-toolbar :global(.search-field) {
			display: block;
			margin-top: 25px;
		}
		.price-head {
			display: none;
		}
		.price-table article {
			display: block;
		}
		.price-table article > span,
		.price-table article > strong {
			display: block;
			margin-top: 18px;
		}
		.method-grid {
			display: block;
		}
		.method-grid article {
			min-height: 260px;
		}
		.method-grid h2 {
			margin-top: 70px;
		}
	}
</style>
