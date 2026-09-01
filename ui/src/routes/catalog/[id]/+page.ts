import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import {
	findBundlesForProduct,
	getProductWithVariants,
} from "$lib/services/stripe";

export const load: PageLoad = async ({ fetch, params }) => {
	const product = await getProductWithVariants(fetch, params.id);

	if (!product) {
		error(404, "Product not found");
	}

	if (product.type === "bundle") {
		const bundle = await findBundlesForProduct(fetch, params.id);

		if (!bundle) error(404, "Bundle not found");

		return {
			product: product,
			containingBundles: bundle,
			title: product.name,
		};
	}

	return {
		product,
		containingBundles: [],
		title: product.name,
	};
};
