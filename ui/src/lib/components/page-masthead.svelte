<script lang="ts">
	let {
		eyebrow,
		title,
		intro,
		image,
		alt,
		index,
	}: {
		eyebrow: string;
		title: string;
		intro: string;
		image: string;
		alt: string;
		index: string;
	} = $props();
</script>

<section class="page-masthead">
	<div class="wrap page-masthead-copy">
		<div class="section-label">
			<span>{index}</span>
			<p>{eyebrow}</p>
		</div>
		<div>
			<h1>{title}</h1>
			<p>{intro}</p>
		</div>
	</div>
	<div class="page-masthead-image"><img src={image} {alt} /></div>
</section>

<style>
	.page-masthead {
		padding-top: 110px;
	}
	.page-masthead-copy {
		display: grid;
		grid-template-columns: 320px 1fr;
		gap: 70px;
		align-items: end;
		padding-bottom: 70px;
	}
	h1 {
		max-width: 980px;
		margin: 0;
		font-family: var(--display);
		font-size: clamp(86px, 9.4vw, 178px);
		font-weight: 800;
		letter-spacing: -0.055em;
		line-height: 0.8;
		text-transform: uppercase;
	}
	.page-masthead-copy > div:last-child > p {
		max-width: 660px;
		margin: 40px 0 0;
		font-size: 18px;
	}
	.page-masthead-image {
		height: 68vh;
		min-height: 580px;
	}
	.page-masthead-image img {
		filter: saturate(0.72);
	}
	@media (max-width: 1000px) {
		.page-masthead-copy {
			grid-template-columns: 220px 1fr;
		}
	}
	@media (max-width: 800px) {
		.page-masthead {
			padding-top: 70px;
		}
		.page-masthead-copy {
			display: block;
		}
		.page-masthead-copy :global(.section-label) {
			margin-bottom: 45px;
		}
		h1 {
			font-size: 70px;
		}
		.page-masthead-copy > div:last-child > p {
			font-size: 15px;
		}
		.page-masthead-image {
			height: 450px;
			min-height: 0;
		}
	}
</style>
