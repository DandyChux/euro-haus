<script lang="ts" module>
	export interface TestimonialItem {
		quote: string;
		name: string;
		title: string;
	}

	export interface LogoItem {
		name: string;
		logoUrl: string;
		link?: string;
	}

	export type InfiniteMovingItem = TestimonialItem | LogoItem;
</script>

<script lang="ts">
	type Direction = 'left' | 'right';
	type Speed = 'fast' | 'normal' | 'slow';
	type Variant = 'testimonial' | 'logo';

	interface Props {
		items: InfiniteMovingItem[];
		direction?: Direction;
		speed?: Speed;
		pauseOnHover?: boolean;
		className?: string;
		variant?: Variant;
	}

	let {
		items,
		direction = 'left',
		speed = 'fast',
		pauseOnHover = true,
		className = '',
		variant = 'testimonial'
	}: Props = $props();

	const durationMap: Record<Speed, string> = {
		fast: '20s',
		normal: '40s',
		slow: '80s'
	};

	let duplicatedItems = $derived([...items, ...items]);
	let animationDuration = $derived(durationMap[speed]);
	let animationDirection = $derived(direction === 'left' ? 'normal' : 'reverse');

	function isLogoItem(item: InfiniteMovingItem): item is LogoItem {
		return 'logoUrl' in item;
	}
</script>

<div
	class={[
		'scroller relative z-20 max-w-7xl overflow-hidden [mask-image:linear-gradient(to_right,transparent,white_20%,white_80%,transparent)]',
		pauseOnHover && 'pause-on-hover',
		className
	]}
	style={`--animation-duration: ${animationDuration}; --animation-direction: ${animationDirection}; --gap: 1rem;`}
>
	<ul class="track">
		{#each duplicatedItems as item, idx (`${isLogoItem(item) ? item.name : item.name}-${idx}`)}
			{#if variant === 'logo' && isLogoItem(item)}
				{@const logo = item}
				<li
					aria-hidden={idx >= items.length}
					class="relative shrink-0 rounded-xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900"
				>
					{#if logo.link}
						<a href={logo.link} target="_blank" rel="noopener noreferrer" class="block">
							<div class="flex h-40 w-60 items-center justify-center p-4">
								<img
									src={logo.logoUrl}
									alt={logo.name}
									class="max-h-full max-w-full object-contain transition-all duration-300 hover:grayscale"
								/>
							</div>
						</a>
					{:else}
						<div class="flex h-40 w-60 items-center justify-center p-4">
							<img
								src={logo.logoUrl}
								alt={logo.name}
								class="max-h-full max-w-full object-contain transition-all duration-300 hover:grayscale"
							/>
						</div>
					{/if}
				</li>
			{:else}
				{@const testimonial = item as TestimonialItem}
				<li
					aria-hidden={idx >= items.length}
					class="relative w-[350px] max-w-full shrink-0 rounded-2xl border border-b-0 border-zinc-200 bg-[linear-gradient(180deg,#fafafa,#f5f5f5)] px-8 py-6 md:w-[450px] dark:border-zinc-700 dark:bg-[linear-gradient(180deg,#27272a,#18181b)]"
				>
					<blockquote>
						<div
							aria-hidden="true"
							class="pointer-events-none absolute -left-0.5 -top-0.5 -z-1 h-[calc(100%_+_4px)] w-[calc(100%_+_4px)] select-none"
						></div>

						<span class="relative z-20 text-sm leading-[1.6] font-normal text-neutral-800 dark:text-gray-100">
							{testimonial.quote}
						</span>

						<div class="relative z-20 mt-6 flex flex-row items-center">
							<span class="flex flex-col gap-1">
								<span class="text-sm leading-[1.6] font-normal text-neutral-500 dark:text-gray-400">
									{testimonial.name}
								</span>
								<span class="text-sm leading-[1.6] font-normal text-neutral-500 dark:text-gray-400">
									{testimonial.title}
								</span>
							</span>
						</div>
					</blockquote>
				</li>
			{/if}
		{/each}
	</ul>
</div>

<style>
	.track {
		display: flex;
		width: max-content;
		min-width: 100%;
		flex-wrap: nowrap;
		gap: var(--gap);
		padding-block: 1rem;
		animation: scroll var(--animation-duration) linear infinite;
		animation-direction: var(--animation-direction);
	}

	.pause-on-hover .track:hover {
		animation-play-state: paused;
	}

	@keyframes scroll {
		to {
			transform: translateX(calc(-50% - var(--gap) / 2));
		}
	}
</style>
