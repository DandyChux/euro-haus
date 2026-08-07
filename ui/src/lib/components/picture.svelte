<script lang="ts">
	// Define the shape of a <source> tag
	type PictureSource = {
		srcset: string;
		media?: string; // e.g. "(min-width: 768px)"
		type?: string; // e.g. "image/webp"
		sizes?: string;
	};

	let {
		src,
		alt,
		sources = [],
		sizes,
		srcset,
		loading = "lazy",
		decoding = "async",
		class: className = "",
		width,
		height,
		...rest
	}: {
		/** The source URL of the image. */
		src: string;
		/** The alternative text description of the image. */
		alt: string;
		/** An array of source objects for the `<picture>` element. */
		sources?: PictureSource[];
		/** The sizes attribute for the `<img> tag`. */
		sizes?: string;
		/** The srcset attribute for the `<img> tag`. */
		srcset?: string;
		/** The loading strategy for the `<img> tag`. */
		loading?: "lazy" | "eager";
		/** The decoding strategy for the `<img> tag`. */
		decoding?: "async" | "sync" | "auto";
		/** CSS classes to apply to the `<img>` tag. */
		class?: string;
		/** The intrinsic width of the image. */
		width?: number | string;
		/** The intrinsic height of the image. */
		height?: number | string;
		/** Any additional attributes to spread onto the `<img>` tag. */
		[key: string]: any;
	} = $props();
</script>

<!--
	We use style="display: contents" on the picture tag so it doesn't
	interfere with CSS Grid/Flexbox layouts. The styling is applied
	directly to the underlying <img> tag.
-->
<picture style="display: contents;">
	{#each sources as source}
		<source
			srcset={source.srcset}
			media={source.media}
			type={source.type}
			sizes={source.sizes || sizes}
		/>
	{/each}

	<img
		{src}
		{alt}
		{srcset}
		{sizes}
		{loading}
		{decoding}
		{width}
		{height}
		class={className}
		{...rest}
	/>
</picture>

<style>
	source {
		display: none !important;
	}
</style>
