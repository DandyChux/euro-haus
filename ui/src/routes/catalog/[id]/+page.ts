import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import {
	findBundlesForProduct,
	getBundleProduct,
	getProductWithVariants,
	getProduct,
	transformStripeProduct,
} from "$lib/services/stripe";

export const load: PageLoad = async ({ fetch, params }) => {
	const raw = await getProduct(fetch, params.id);

	if (!raw) {
		error(404, "Product not found");
	}

	if (raw.metadata.type === "bundle") {
		const bundle = await getBundleProduct(fetch, params.id);

		if (!bundle) error(404, "Bundle not found");

		return {
			product: bundle,
			containingBundles: [],
			title: bundle.title,
		};
	}

	const product =
		(await getProductWithVariants(fetch, params.id)) ??
		transformStripeProduct(raw);

	return {
		product,
		containingBundles: await findBundlesForProduct(fetch, params.id),
		title: product.title,
	};
};
